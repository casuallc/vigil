#!/bin/bash
# ===================================================================
# Application Management Script - appctl
# Purpose: Manage application lifecycle (start, stop, restart, status, logs, service install)
# Usage: ./appctl {start|stop|restart|status|logs|install-service}
# Requirements: POSIX shell, systemd (for service install)
# ===================================================================

# PROGRAM directory
BINDIR=$(dirname "$0")
APP_HOME=`cd -P $BINDIR/..;pwd`
cd $APP_HOME
CONFIG_FILE="$APP_HOME/conf/app.conf"

# Load configuration if exists
if [[ -f "$CONFIG_FILE" ]]; then
    source "$CONFIG_FILE"
fi

# Default configuration (can be overridden in app.conf)
APP_NAME="${APP_NAME:-MyApp}"
APP_BINARY="${APP_BINARY:-$APP_NAME}"
APP_ARGS="${APP_ARGS:-}"
LOG_FILE="${LOG_FILE:-$APP_HOME/logs/app.log}"
PID_FILE="${PID_FILE:-$APP_HOME/app.pid}"
USER="${USER:-$(whoami)}"
ENV_FILE="${ENV_FILE:-$APP_HOME/.env}"
RUN_DIR="${RUN_DIR:-$APP_HOME}"

# Ensure log directory exists
LOG_DIR="$(dirname "$LOG_FILE")"
mkdir -p "$LOG_DIR"

# Function: Load environment variables
load_env() {
    if [[ -f "$ENV_FILE" ]]; then
        export $(grep -v '^#' "$ENV_FILE" | xargs)
    fi
}

# Function: Check if app is running
is_running() {
    if [[ -f "$PID_FILE" ]]; then
        PID=$(cat "$PID_FILE" 2>/dev/null)
        if kill -0 "$PID" 2>/dev/null; then
            return 0
        else
            rm -f "$PID_FILE"
            return 1
        fi
    fi
    return 1
}

# Function: Start application
start() {
    if is_running; then
        PID=$(cat "$PID_FILE")
        echo "$APP_NAME is already running, PID: $PID"
        return 1
    fi

    load_env

    # Validate binary
    if [[ ! -x "$APP_BINARY" ]] && ! command -v "$APP_BINARY" >/dev/null 2>&1; then
        echo "ERROR: Binary not found or not executable: $APP_BINARY" | tee -a "$LOG_FILE"
        exit 1
    fi

    # Ensure log file is writable
    touch "$LOG_FILE" 2>/dev/null || {
        echo "ERROR: Cannot write to log file: $LOG_FILE" | tee -a "$LOG_FILE"
        exit 1
    }

    cd "$RUN_DIR" || {
        echo "ERROR: Cannot change to working directory: $RUN_DIR" | tee -a "$LOG_FILE"
        exit 1
    }

    case "${1:-}" in
        --foreground|fg)
            echo "Starting $APP_NAME in foreground mode..."
            echo "Logging to: $LOG_FILE"
            exec env ${APP_ENV} "$APP_BINARY" $APP_ARGS
            ;;
        *)
            echo "Starting $APP_NAME in background mode..." >> "$LOG_FILE"
            echo "Launch time: $(date)" >> "$LOG_FILE"
            echo "Command: env ${APP_ENV} $APP_BINARY $APP_ARGS" >> "$LOG_FILE"

            # Start in background with full logging
            (
                echo "=== $APP_NAME Startup (PID: \$\$) ==="
                echo "Time: $(date)"
                echo "Command: env ${APP_ENV} $APP_BINARY $APP_ARGS"
                echo "Working Directory: $RUN_DIR"
                exec env ${APP_ENV} "$APP_BINARY" $APP_ARGS
            ) >> "$LOG_FILE" 2>&1 &

            APP_PID=$!
            echo $APP_PID > "$PID_FILE"
            chmod 644 "$PID_FILE"

            # Wait and check if proc is still alive
            sleep 3

            if kill -0 $APP_PID 2>/dev/null; then
                echo "$APP_NAME started successfully, PID: $APP_PID"
                echo "Status: RUNNING" >> "$LOG_FILE"
            else
                # Process died quickly → startup failed
                rm -f "$PID_FILE"
                echo "ERROR: $APP_NAME started but exited immediately (PID: $APP_PID). Check logs." >&2
                echo "Status: FAILED (crashed at startup)" >> "$LOG_FILE"
                return 1
            fi
            ;;
    esac
}

# Function: Stop application
stop() {
    if is_running; then
        PID=$(cat "$PID_FILE")
        echo "Stopping $APP_NAME (PID: $PID)..."
        kill "$PID" 2>/dev/null
        for i in {1..10}; do
            if kill -0 "$PID" 2>/dev/null; then
                sleep 1
            else
                rm -f "$PID_FILE"
                echo "$APP_NAME stopped."
                return 0
            fi
        done
        echo "Process did not respond, forcing termination..."
        kill -9 "$PID" 2>/dev/null
        rm -f "$PID_FILE"
        echo "$APP_NAME stopped (forced)."
    else
        echo "$APP_NAME is not running."
    fi
}

# Function: Restart application
restart() {
    stop
    sleep 2
    start
}

# Function: Show application status
status() {
    if is_running; then
        PID=$(cat "$PID_FILE")
        echo "$APP_NAME is running, PID: $PID"
    else
        echo "$APP_NAME is not running."
    fi
}

# Function: Show logs
logs() {
    if [[ -f "$LOG_FILE" ]]; then
        tail -f "$LOG_FILE"
    else
        echo "Log file $LOG_FILE does not exist."
    fi
}

# Function: Install as systemd service (requires root)
install_service() {
    SERVICE_NAME="$APP_NAME.service"
    SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME"

    if [[ $(id -u) -ne 0 ]]; then
        echo "Error: Installing system service requires root privileges."
        exit 1
    fi

    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=$APP_NAME Service
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$RUN_DIR
EnvironmentFile=$ENV_FILE
ExecStart=$APP_HOME/bin/appctl --foreground
Restart=always
StandardOutput=append:$LOG_FILE
StandardError=append:$LOG_FILE

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "$SERVICE_FILE"
    systemctl daemon-reload
    echo "Service installed: $SERVICE_NAME"
    echo "To enable and start, run:"
    echo "  systemctl enable $SERVICE_NAME"
    echo "  systemctl start $SERVICE_NAME"
}

# Main command dispatcher
case "${1:-}" in
    start)
        start "$2"
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    install-service)
        install_service
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|install-service}"
        echo "  start           - Start in background"
        echo "  start fg        - Start in foreground"
        echo "  stop            - Stop the application"
        echo "  restart         - Restart the application"
        echo "  status          - Show running status"
        echo "  logs            - Tail the log file"
        echo "  install-service - Install as systemd service (requires root)"
        exit 1
        ;;
esac

exit 0