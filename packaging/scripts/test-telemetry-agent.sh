#!/bin/bash

# Detect OS and version
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VERSION_ID=$(echo $VERSION_ID | cut -d'.' -f1)
else
    echo "Unsupported OS"
    exit 1
fi

TARGET_TEST_VERSION="$1"

remove_percona_telemetry() {
    echo "Checking if Percona telemetry agent is installed..."

    case "$OS" in
        ol | amzn)
            # Oracle Linux and Amazon Linux
            if rpm -q percona-telemetry-agent; then
                echo "Percona telemetry agent is installed. Removing..."
                yum remove -y percona-telemetry-agent
                echo "Removing Percona repository files..."
                rm -f /etc/yum.repos.d/percona-*.repo
            else
                echo "Percona telemetry agent is not installed."
            fi
            ;;
        debian | ubuntu)
            if dpkg -l | grep -q percona-telemetry-agent; then
                echo "Percona telemetry agent is installed. Removing..."
                apt-get remove -y percona-telemetry-agent
                echo "Removing Percona repository files..."
                rm -f /etc/apt/sources.list.d/percona-*.list
                apt-get update
            else
                echo "Percona telemetry agent is not installed."
            fi
            ;;
        *)
            echo "Unsupported OS: ${OS}"
            exit 1
            ;;
    esac
}

install_percona_release() {
  case "$OS" in
          ol)
              # Oracle Linux
              if [ "$VERSION_ID" == "8" ] || [ "$VERSION_ID" == "9" ]; then
                  yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
              else
                  echo "Unsupported Oracle Linux version: ${VERSION_ID}"
                  exit 1
              fi
              ;;
          amzn)
            # Amazon Linux
            if [ "$VERSION_ID" == "2" ] || [ "$VERSION_ID" == "2023" ]; then
              # yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
              # todo: use percona-release from testing repositories until it's released
              yum -y install https://repo.percona.com/prel/yum/testing/2023/RPMS/noarch/percona-release-1.0-30.noarch.rpm
            else
              echo "Unsupported Amazon Linux version: ${VERSION_ID}"
              exit 1
            fi
            ;;
          debian | ubuntu)
              if [ "$VERSION_ID" == "11" ] || [ "$VERSION_ID" == "12" ] || [ "$VERSION_ID" == "20" ] || [ "$VERSION_ID" == "22" ] || [ "$VERSION_ID" == "24" ]; then
                  apt-get update
                  apt-get install -y wget gnupg2 lsb-release curl systemd
                  wget https://repo.percona.com/apt/percona-release_latest.$(lsb_release -sc)_all.deb
                  dpkg -i percona-release_latest.$(lsb_release -sc)_all.deb
              else
                  echo "Unsupported Debian/Ubuntu version: ${VERSION_ID}"
                  exit 1
              fi
              ;;
          *)
              echo "Unsupported OS"
              exit 1
              ;;
      esac
}

check_telemetry_agent_logs() {
  if [ ! -d /var/log/percona/telemetry-agent ]; then
      echo "telemetry-agent log location is missing"
      exit 1
  fi

  if [ ! -f /var/log/percona/telemetry-agent/telemetry-agent.log ]; then
      echo "telemetry-agent log file is missing"
      exit 1
  fi

  if [ ! -f /var/log/percona/telemetry-agent/telemetry-agent-error.log ]; then
      echo "telemetry-agent error log file is missing"
      exit 1
  fi

  echo "validated telemetry-agent log files"
}

check_percona_telemetry_version() {
  output=$(percona-telemetry-agent --version)

  version=$(echo "$output" | grep "Version:" | awk '{print $2}')
  commit=$(echo "$output" | grep "Commit:" | awk '{print $2}')
  build_date=$(echo "$output" | grep "Build date:" | awk '{print $3}')

  if [ -z "$version" ]; then
      echo "Error: Version information is empty"
      exit 1
  fi

  if [ ! -z "$TARGET_TEST_VERSION" ]; then
    if [ "$version" != "$TARGET_TEST_VERSION" ]; then
      echo "Error: Build version ($version) does not match expected version ($TARGET_TEST_VERSION)"
      exit 1
    fi
  fi

  if [ -z "$commit" ]; then
      echo "Error: Commit information is empty"
      exit 1
  fi

  if [ -z "$build_date" ]; then
      echo "Error: Build date information is empty"
      exit 1
  fi

  echo "Version: $version"
  echo "Commit: $commit"
  echo "Build date: $build_date"
}

test_percona_telemetry_installation() {
    # Call remove function to clean the system before installation
    remove_percona_telemetry

    # install percona-release
    install_percona_release
    percona-release enable telemetry testing

    if [ "$OS" == "ol" ] || [ "$OS" == "amzn" ]; then
        yum install -y percona-telemetry-agent
    else
        apt-get update
        apt-get install -y percona-telemetry-agent
    fi

    # Check version info for the installed telemetry-agent
    check_percona_telemetry_version

    # Check telemetry-agent logs
    check_telemetry_agent_logs

    systemctl is-enabled percona-telemetry-agent
    if [ $? -eq 0 ]; then
        echo "Service is enabled as expected."
    else
        echo "Warning: Service is disabled, but it should be enabled post installation."
        exit 1
    fi

    systemctl is-active percona-telemetry-agent
    if [ $? -eq 0 ]; then
        echo "Service is running as expected."
    else
        echo "Warning: Service is inactive, but it should be active post installation."
        exit 1
    fi

    # Clean up
    remove_percona_telemetry
}

# tests that updating percona-telemetry-agent works.
# it also accepts argument to determine if TA should be enabled or disabled before updating.
test_percona_telemetry_update() {
  remove_percona_telemetry

  install_percona_release

  if [ "$OS" == "ol" ]; then
    # enable and install from the main repository so that we can update from that to the testing package.
    percona-release enable telemetry
    yum install -y percona-telemetry-agent
  elif [ "$OS" == "amzn" ]; then
    # install from testing repo until we publish to main
    percona-release enable telemetry testing
    yum install -y percona-telemetry-agent
  else
    # enable and install from the main repository so that we can update from that to the testing package.
    percona-release enable telemetry
    apt-get update
    apt-get install -y percona-telemetry-agent
  fi

  check_percona_telemetry_version
  check_telemetry_agent_logs

  if [ "$1" == "disabled" ]; then
    systemctl stop percona-telemetry-agent
    systemctl disable percona-telemetry-agent
  else
    systemctl enable percona-telemetry-agent
  fi

  pre_update_status=""
  systemctl is-enabled percona-telemetry-agent | grep -q "enabled"
  if [ $? -eq 0 ]; then
    pre_update_status="enabled"
  else
    pre_update_status="disabled"
  fi
  echo "telemetry-agent status before update is $pre_update_status"

  # upgrade TA and recheck
  percona-release enable telemetry testing
  if [ "$OS" == "ol" ] || [ "$OS" == "amzn" ]; then
      yum update -y percona-telemetry-agent
  else
      apt-get update
      apt-get install --only-upgrade -y percona-telemetry-agent
  fi

  check_percona_telemetry_version
  check_telemetry_agent_logs
  systemctl is-enabled percona-telemetry-agent | grep -q $pre_update_status
  if [ $? -eq 0 ]; then
    echo "telemetry-agent status remained $pre_update_status after update"
  else
    echo "telemetry-agent status after update was not $pre_update_status"
    exit 1
  fi

  # clean up
  remove_percona_telemetry
}

test_percona_telemetry_installation
test_percona_telemetry_update "enabled"
test_percona_telemetry_update "disabled"
