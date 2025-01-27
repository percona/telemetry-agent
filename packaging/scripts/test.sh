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

remove_percona_telemetry() {
    echo "Checking if Percona telemetry agent is installed..."

    case "$OS" in
        ol | amzn)
            # Oracle Linux
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
            echo "Unsupported OS"
            exit 1
            ;;
    esac
}

install_percona_telemetry() {

    # Call remove function to clean the system before installation
    remove_percona_telemetry

    case "$OS" in
        ol)
            # Oracle Linux
            if [ "$VERSION_ID" == "8" ] || [ "$VERSION_ID" == "9" ]; then
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
            else
                echo "Unsupported Oracle Linux version"
                exit 1
            fi
            ;;
        amzn)
          # Amazon Linux
          if [ "$VERSION_ID" == "2023" ]; then
            yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
          else
            echo "Unsupported Amazon Linux version"
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
                echo "Unsupported Debian/Ubuntu version"
                exit 1
            fi
            ;;
        *)
            echo "Unsupported OS"
            exit 1
            ;;
    esac

    percona-release enable telemetry

    if [ "$OS" == "ol" ]; then
        yum install -y percona-telemetry-agent
    else
        apt-get update
        apt-get install -y percona-telemetry-agent
    fi

    systemctl stop percona-telemetry-agent
    systemctl disable percona-telemetry-agent

    percona-release enable telemetry testing

    if [ "$OS" == "ol" ] || [ "$OS" == "amzn" ]; then
        yum update -y percona-telemetry-agent
    else
        apt-get update
        apt-get install --only-upgrade -y percona-telemetry-agent
    fi

    systemctl is-enabled percona-telemetry-agent | grep -q "disabled"
    if [ $? -eq 0 ]; then
        echo "Service is still disabled as expected."
    else
        echo "Warning: Service is enabled, but it should be disabled."
        exit 1
    fi
}

# Start installation process
install_percona_telemetry
