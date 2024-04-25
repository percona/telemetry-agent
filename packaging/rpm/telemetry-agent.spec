%global debug_package %{nil}
%define _log_dir /var/log/percona

Name:  percona-telemetry-agent
Version: @@VERSION@@
Release: 1%{?dist}
Summary: Percona Telemetry Agent
Group:  Applications/Databases
License: GPLv3
URL:  https://github.com/percona/telemetry-agent
Source0: percona-telemetry-agent-%{version}.tar.gz

BuildRequires: golang make git
BuildRequires:  systemd
BuildRequires:  pkgconfig(systemd)
Requires(post):   systemd
Requires(preun):  systemd
Requires(postun): systemd

%description
Percona Telemetry Agent gathers information and metrics from Percona products installed on the host.

%prep
%setup -q -n percona-telemetry-agent-%{version}


%build
source ./VERSION
export VERSION
export GITBRANCH
export GITCOMMIT

cd ../
export PATH=/usr/local/go/bin:${PATH}
export GOROOT="/usr/local/go/"
export GOPATH=$(pwd)/
export PATH="/usr/local/go/bin:$PATH:$GOPATH"
export GOBINPATH="/usr/local/go/bin"
mkdir -p src/github.com/percona/
mv percona-telemetry-agent-%{version} src/github.com/percona/percona-telemetry-agent
ln -s src/github.com/percona/percona-telemetry-agent percona-telemetry-agent-%{version}
cd src/github.com/percona/percona-telemetry-agent && make build
cd %{_builddir}

%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/%{_log_dir}
install -m 755 -d $RPM_BUILD_ROOT/%{_bindir}
cd ../
export PATH=/usr/local/go/bin:${PATH}
export GOROOT="/usr/local/go/"
export GOPATH=$(pwd)/
export PATH="/usr/local/go/bin:$PATH:$GOPATH"
export GOBINPATH="/usr/local/go/bin"
cd src/
cp github.com/percona/percona-telemetry-agent/bin/telemetry-agent $RPM_BUILD_ROOT/%{_bindir}/percona-telemetry-agent
install -m 0755 -d $RPM_BUILD_ROOT/%{_sysconfdir}
install -D -m 0644 github.com/percona/percona-telemetry-agent/packaging/conf/percona-telemetry-agent.logrotate $RPM_BUILD_ROOT/%{_sysconfdir}/logrotate.d/percona-telemetry-agent
install -m 0755 -d $RPM_BUILD_ROOT/%{_sysconfdir}/sysconfig
install -D -m 0640 github.com/percona/percona-telemetry-agent/packaging/conf/percona-telemetry-agent.env $RPM_BUILD_ROOT/%{_sysconfdir}/sysconfig/percona-telemetry-agent
install -m 0755 -d $RPM_BUILD_ROOT/%{_unitdir}
install -m 0644 github.com/percona/percona-telemetry-agent/packaging/conf/percona-telemetry-agent.service $RPM_BUILD_ROOT/%{_unitdir}/percona-telemetry-agent.service

%pre -n percona-telemetry-agent
if [ ! -d /run/percona-telemetry-agent ]; then
    install -m 0755 -d -oroot -groot /run/percona-telemetry-agent
fi
# Create new linux group
groupadd percona-telemetry
# For telemetry-agent to be able to read/remove the metric files
usermod -a -G percona-telemetry daemon

%post -n percona-telemetry-agent
%systemd_post percona-telemetry-agent.service
/usr/bin/systemctl enable percona-telemetry-agent >/dev/null 2>&1 || :
/usr/bin/systemctl start percona-telemetry-agent
# Create telemetry history directory
mkdir -p /usr/local/percona/telemetry/history
chown daemon:percona-telemetry /usr/local/percona/telemetry/history
chmod g+s /usr/local/percona/telemetry/history
chmod u+s /usr/local/percona/telemetry/history
chown daemon:percona-telemetry /usr/local/percona/telemetry
# Fix permissions to be able to create Percona telemetry uuid file
chgrp percona-telemetry /usr/local/percona
chmod 775 /usr/local/percona

%preun -n percona-telemetry-agent
/usr/bin/systemctl stop percona-telemetry-agent || true

%postun -n percona-telemetry-agent
%systemd_postun_with_restart percona-telemetry-agent.service

%files -n percona-telemetry-agent
%{_bindir}/percona-telemetry-agent
%dir %attr(0755,root,root) %{_log_dir}
%config(noreplace) %attr(0640,root,root) /%{_sysconfdir}/sysconfig/percona-telemetry-agent
%config(noreplace) %attr(0644,root,root) /%{_sysconfdir}/logrotate.d/percona-telemetry-agent
%{_unitdir}/percona-telemetry-agent.service

%changelog
* Wed Apr 03 2024 Surabhi Bhat <surabhi.bhat@percona.com>
- First build
