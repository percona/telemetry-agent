%global debug_package %{nil}
%define _log_dir /var/log/percona/telemetry-agent

Name:  percona-telemetry-agent
Version: @@VERSION@@
Release: @@RELEASE@@%{?dist}
Summary: Percona Telemetry Agent
Group:  Applications/Databases
License: GPLv3
URL:  https://github.com/percona/telemetry-agent
Source0: percona-telemetry-agent-%{version}.tar.gz

BuildRequires: golang make git
BuildRequires:  systemd
BuildRequires:  pkgconfig(systemd)
Requires:  logrotate
Requires(post):   systemd
Requires(preun):  systemd
Requires(postun): systemd
%if 0%{?rhel} <= 7
Requires:  yum-utils
%endif

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
%ifarch aarch64
export GOARCH=arm64
%else
export GOARCH=amd64
%endif
mkdir -p src/github.com/percona/
mv percona-telemetry-agent-%{version} src/github.com/percona/percona-telemetry-agent
ln -s src/github.com/percona/percona-telemetry-agent percona-telemetry-agent-%{version}
cd src/github.com/percona/percona-telemetry-agent
export GO111MODULE=on
export GOMODCACHE=$(pwd)/go-mod-cache
for i in {1..3}; do
    go mod tidy && go mod download && break
    echo "go mod commands failed, retrying in 10 seconds..."
    sleep 10
done
env GOARCH=${GOARCH} make build
cd %{_builddir}

%install
rm -rf $RPM_BUILD_ROOT
install -m 755 -d $RPM_BUILD_ROOT/%{_bindir}
install -m 0775 -d $RPM_BUILD_ROOT/%{_log_dir}
install -D -m 0660 /dev/null $RPM_BUILD_ROOT/%{_log_dir}/telemetry-agent.log
install -D -m 0660 /dev/null  $RPM_BUILD_ROOT/%{_log_dir}/telemetry-agent-error.log
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
# For telemetry-agent to be able to read/remove the metric files
/usr/bin/getent group percona-telemetry || groupadd percona-telemetry >/dev/null 2>&1 || :
usermod -a -G percona-telemetry daemon >/dev/null 2>&1 || :

%post -n percona-telemetry-agent
chown -R daemon:percona-telemetry %{_log_dir} >/dev/null 2>&1 || :
chmod g+w %{_log_dir}
# Move the old logfiles, if present during update
if ls /var/log/percona/telemetry-agent*log* >/dev/null 2>&1; then
    chmod 0775  %{_log_dir}
    mv /var/log/percona/telemetry-agent*log* /var/log/percona/telemetry-agent/ >/dev/null 2>&1 || :
    chmod 0660  %{_log_dir}/telemetry-agent*log*
fi
# Create telemetry history directory
mkdir -p /usr/local/percona/telemetry/history
chown daemon:percona-telemetry /usr/local/percona/telemetry/history
chmod g+s /usr/local/percona/telemetry/history
chmod u+s /usr/local/percona/telemetry/history
chown daemon:percona-telemetry /usr/local/percona/telemetry
# Fix permissions to be able to create Percona telemetry uuid file
chgrp percona-telemetry /usr/local/percona
chmod 775 /usr/local/percona
%systemd_post percona-telemetry-agent.service
if [ $1 == 1 ]; then
      /usr/bin/systemctl enable percona-telemetry-agent >/dev/null 2>&1 || :
fi

%preun -n percona-telemetry-agent
%systemd_preun percona-telemetry-agent.service

%postun -n percona-telemetry-agent
if [ $1 == 0 ]; then
    %systemd_postun_with_restart percona-telemetry-agent.service
    systemctl daemon-reload
    groupdel percona-telemetry >/dev/null 2>&1 || :
fi

%posttrans -n percona-telemetry-agent
# Package update - add the group that was deleted, reload and restart the service
if [ $1 -ge 1 ]; then
    /usr/bin/getent group percona-telemetry || groupadd percona-telemetry >/dev/null 2>&1 || :
    usermod -a -G percona-telemetry daemon >/dev/null 2>&1 || :
    systemctl daemon-reload >/dev/null 2>&1 || true
    if systemctl is-enabled percona-telemetry-agent.service > /dev/null 2>&1; then
        #/usr/bin/systemctl enable percona-telemetry-agent.service >/dev/null 2>&1 || :
        /usr/bin/systemctl restart percona-telemetry-agent.service >/dev/null 2>&1 || :
    fi
fi

%files -n percona-telemetry-agent
%{_bindir}/percona-telemetry-agent
%config(noreplace) %attr(0640,root,root) /%{_sysconfdir}/sysconfig/percona-telemetry-agent
%config(noreplace) %attr(0644,root,root) /%{_sysconfdir}/logrotate.d/percona-telemetry-agent
%{_unitdir}/percona-telemetry-agent.service
%{_log_dir}/telemetry-agent.log
%{_log_dir}/telemetry-agent-error.log

%changelog
* Wed Apr 03 2024 Surabhi Bhat <surabhi.bhat@percona.com>
- First build
