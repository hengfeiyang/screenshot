package main

import (
  "crypto/md5"
  "encoding/hex"
  "errors"
  "flag"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "regexp"
  "runtime"
  "strconv"
  "syscall"
)

const (
  DEF_PORT   int  = 9464
  DEF_DELAY  int  = 1
  DEF_WIDTH  int  = 1024
  DEF_HEIGHT int  = 768
  DEF_DEBUG  bool = false

  BASE_DIR  string = "/usr/local/screenShot" // 程序工作目录
  PHANTOMJS string = "/usr/bin/phantomjs"    // PhantomJS执行路径
  SNAP_JS   string = "util/rasterize.js"     // 截图脚本路径
)

type config struct {
  dir    string // 程序工作目录
  port   int    // screenShot服务端口
  delay  int    // 截图延迟
  width  int    // 屏幕宽度
  height int    // 屏幕高度
  debug  bool   // 是否开启调试模式
}

var conf *config // 创建全局变量 conf

func main() {
  // 初始化配置
  var pidfile string
  var isDaemon bool
  conf = new(config)
  flag.StringVar(&pidfile, "pidfile", "", "pid file")
  flag.BoolVar(&isDaemon, "daemon", false, "as a daemon service (default: false)")
  flag.StringVar(&conf.dir, "datdir", BASE_DIR, "util script and data directory (default: "+BASE_DIR+")")
  flag.IntVar(&conf.port, "port", DEF_PORT, "TCP port number to listen on (default: "+strconv.Itoa(DEF_PORT)+")")
  flag.IntVar(&conf.delay, "delay", DEF_DELAY, "Delay second before shot (default: "+strconv.Itoa(DEF_DELAY)+")")
  flag.IntVar(&conf.width, "width", DEF_WIDTH, "Screen width (default: "+strconv.Itoa(DEF_WIDTH)+")")
  flag.IntVar(&conf.height, "height", DEF_HEIGHT, "Screen height (default: "+strconv.Itoa(DEF_HEIGHT)+")")
  flag.BoolVar(&conf.debug, "debug", DEF_DEBUG, "Open debug mode (default: false)")
  flag.Parse()
  //daemon
  if isDaemon == true && len(pidfile) > 4 {
   daemon(1, 1)
   pid := os.Getpid()
   err := ioutil.WriteFile(pidfile, []byte(strconv.Itoa(pid)), 0666)
   if err != nil {
    fmt.Println(err.Error())
    os.Exit(1)
   }
  }
  //为phantomJS配置环境变量
  os.Setenv("LIBXCB_ALLOW_SLOPPY_LOCK", "1")
  os.Setenv("DISPLAY", ":0")
  // 开始服务
  http.HandleFunc("/", handler)
  log.Fatalln("ListenAndServe: ", http.ListenAndServe(":"+strconv.Itoa(conf.port), nil))
}

// 处理请求
func handler(rw http.ResponseWriter, req *http.Request) {
  rw.Header().Set("Server", "GWS")
  var url = req.FormValue("url")
  var width = req.FormValue("width")
  var height = req.FormValue("height")
  var delay = req.FormValue("delay")
  var flush = req.FormValue("flush")
  var validURL = regexp.MustCompile(`^http(s)?://.*$`)
  if ok := validURL.MatchString(url); !ok {
   fmt.Fprintf(rw, "<html><body>请输入需要截图的网址：<form><input name=url><input type=submit value=shot></form></body></html>")
  } else {
   pic := GetPicPath(url)
   // 如果有range,表明是分段请求，直接处理
   if v := req.Header.Get("Range"); v != "" {
    http.ServeFile(rw, req, pic)
   }
   // 判断图片是否重新生成
   if i, _ := strconv.Atoi(flush); i == 1 || IsExist(pic) == false {
    pic, err := exec(url, pic, width, height, delay)
    if err != nil {
      if conf.debug == true {
        log.Println("Snapshot Error:", url, err.Error())
      }
      fmt.Fprintf(rw, "shot error: %s", err.Error())
      return
    }
    if conf.debug == true {
      log.Println("Snapshot Successful:", url, pic)
    }
   }
   http.ServeFile(rw, req, pic)
  }
  return
}

// 执行截图
func exec(url, pic, width, height, delay string) (string, error) {
  if url == "" {
   return "", errors.New("url is none.")
  }
  if width == "" {
   width = strconv.Itoa(conf.width)
  }
  if height == "" {
   height = strconv.Itoa(conf.height)
  }
  if delay == "" {
   delay = strconv.Itoa(conf.delay)
  }
  procAttr := new(os.ProcAttr)
  procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
  procAttr.Dir = os.Getenv("PWD")
  procAttr.Env = os.Environ()
  var args []string
  args = make([]string, 7)
  args[0] = PHANTOMJS
  args[1] = conf.dir + "/" + SNAP_JS
  args[2] = url
  args[3] = pic
  args[4] = delay
  args[5] = width
  args[6] = height
  process, err := os.StartProcess(PHANTOMJS, args, procAttr)
  if err != nil {
   if conf.debug == true {
    log.Println("PhantomJS start failed:" + err.Error())
   }
   return "", err
  }
  waitMsg, err := process.Wait()
  if err != nil {
   if conf.debug == true {
    log.Println("PhantomJS start wait error:" + err.Error())
   }
   return "", err
  }
  if conf.debug == true {
   log.Println(waitMsg)
  }
  return args[3], nil
}

// 根据url获取图片路径
func GetPicPath(url string) string {
  h := md5.New()
  h.Write([]byte(url))
  pic := hex.EncodeToString(h.Sum(nil))
  path := conf.dir + "/data/"
  os.Mkdir(path, 0755)
  path += string(pic[0:2]) + "/"
  os.Mkdir(path, 0755)
  return path + pic + ".png"
}

// 判断一个文件或目录是否存在
func IsExist(path string) bool {
  _, err := os.Stat(path)
  if err == nil {
   return true
  }
  // Check if error is "no such file or directory"
  if _, ok := err.(*os.PathError); ok {
   return false
  }
  return false
}

// deamon
func daemon(nochdir, noclose int) int {
  var ret, ret2 uintptr
  var err syscall.Errno

  darwin := runtime.GOOS == "darwin"

  // already a daemon
  if syscall.Getppid() == 1 {
   return 0
  }

  // fork off the parent process
  ret, ret2, err = syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
  if err != 0 {
   return -1
  }

  // failure
  if ret2 < 0 {
   os.Exit(-1)
  }

  // handle exception for darwin
  if darwin && ret2 == 1 {
   ret = 0
  }

  // if we got a good PID, then we call exit the parent process.
  if ret > 0 {
   os.Exit(0)
  }

  /* Change the file mode mask */
  _ = syscall.Umask(0)

  // create a new SID for the child process
  s_ret, s_errno := syscall.Setsid()
  if s_errno != nil {
   log.Printf("Error: syscall.Setsid errno: %d", s_errno)
  }
  if s_ret < 0 {
   return -1
  }

  if nochdir == 0 {
   os.Chdir("/")
  }

  if noclose == 0 {
   f, e := os.OpenFile("/dev/null", os.O_RDWR, 0)
   if e == nil {
    fd := f.Fd()
    syscall.Dup2(int(fd), int(os.Stdin.Fd()))
    syscall.Dup2(int(fd), int(os.Stdout.Fd()))
    syscall.Dup2(int(fd), int(os.Stderr.Fd()))
   }
  }

  return 0
}
