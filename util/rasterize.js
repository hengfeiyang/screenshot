var page = require('webpage').create(),
    system = require('system'),
    address, output, size,
    delay = 200,
    width = 960,
    height = 600;

if (system.args.length < 3 || system.args.length > 8) {
    console.log('Usage: rasterize.js URL filename [delay] [viewWidth] [viewHeight] [paperwidth*paperheight|paperformat] [zoom]');
    console.log('  paper (pdf output) examples: "5in*7.5in", "10cm*20cm", "A4", "Letter"');
    phantom.exit(1);
} else {
    address = system.args[1];
    output = system.args[2];
    if (system.args.length > 4) width = system.args[4];
    if (system.args.length > 5) height = system.args[5];
    page.viewportSize = { width: width, height: height };
    if (system.args.length > 6 && system.args[2].substr(-4) === ".pdf") {
        size = system.args[6].split('*');
        page.paperSize = size.length === 2 ? { width: size[0], height: size[1], margin: '0px' }
                                           : { format: system.args[6], orientation: 'portrait', margin: '1cm' };
    }
    if (system.args.length > 7) page.zoomFactor = system.args[7];
    if (system.args.length > 3) delay = system.args[3] * 1000;
    page.open(address, function (status) {
        if (status !== 'success') {
            console.log('Unable to load the address!');
            phantom.exit();
        } else {
            window.setTimeout(function () {
                page.render(output);
                phantom.exit();
            }, delay);
        }
    });
}
