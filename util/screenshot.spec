Summary: screenshot is a http service for screen Shot based on PhantomJS.
Name: screenshot
Version: 1.0.0
Release: 1%{?dist}
Vendor: Coldstar
URL: https://github.com/9466/screenshot/
License: BSD
BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root
BuildRequires: glibc
BuildRequires: phantomjs
Provides: webserver

Source: %{name}-%{version}.tar.gz

%description
screenshot is a http service for screen Shot based on PhantomJS.

%prep
%setup -q

%install
%{__mkdir} -p %{buildroot}
%{__cp} -R * %{buildroot}/
%{__mkdir} -p %{buildroot}/etc/rc.d/init.d
mv %{buildroot}/usr/local/screenshot/util/screenshot.init %{buildroot}/etc/rc.d/init.d/screenshot

%clean
%{__rm} -rf $RPM_BUILD_ROOT

%files
%defattr(-,root,root)
%{_bindir}/screenshot
%{_initddir}/screenshot
%{_usr}/local/screenshot/data
%{_usr}/local/screenshot/util
%{_usr}/local/screenshot/util/rasterize.js

%pre

%post
# Register the nginx service
if [ $1 -eq 1 ]; then
    /sbin/chkconfig --add screenshot
fi

%preun
if [ $1 -eq 0 ]; then
    /sbin/service screenshot stop > /dev/null 2>&1
    /sbin/chkconfig --del screenshot
fi

%postun
if [ $1 -ge 1 ]; then
    /sbin/service screenshot restart &>/dev/null || :
fi
