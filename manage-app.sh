#!/bin/bash
# Axiom IDP - Application Management Script
# Start, stop, and manage the local development environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="$SCRIPT_DIR/axiom.log"
BACKEND_PID_FILE="$SCRIPT_DIR/.axiom-backend.pid"
FRONTEND_PID_FILE="$SCRIPT_DIR/.axiom-frontend.pid"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Start backend
start_backend() {
    log "Starting backend server..."
    
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            error "Backend is already running (PID: $pid)"
            return 1
        fi
    fi
    
    cd "$SCRIPT_DIR"
    nohup ./bin/axiom-server >> "$LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$BACKEND_PID_FILE"
    
    sleep 2
    
    if ps -p "$pid" > /dev/null 2>&1; then
        success "Backend started (PID: $pid)"
        success "Backend running on http://localhost:8080"
        success "Health check: http://localhost:8080/health"
        return 0
    else
        error "Failed to start backend"
        return 1
    fi
}

# Start frontend
start_frontend() {
    log "Starting frontend server..."
    
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            error "Frontend is already running (PID: $pid)"
            return 1
        fi
    fi
    
    cd "$SCRIPT_DIR/web"
    nohup python3 -m http.server 3000 --directory dist >> "$LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$FRONTEND_PID_FILE"
    
    sleep 1
    
    if ps -p "$pid" > /dev/null 2>&1; then
        success "Frontend started (PID: $pid)"
        success "Frontend running on http://localhost:3000"
        return 0
    else
        error "Failed to start frontend"
        return 1
    fi
}

# Start both
start_all() {
    log "Starting Axiom IDP..."
    echo ""
    
    start_backend
    echo ""
    start_frontend
    
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Axiom IDP is running!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Access the application:"
    echo "  Web UI:     ${BLUE}http://localhost:3000${NC}"
    echo "  API:        ${BLUE}http://localhost:8080${NC}"
    echo "  Health:     ${BLUE}http://localhost:8080/health${NC}"
    echo ""
    echo "View logs:"
    echo "  ${BLUE}tail -f $LOG_FILE${NC}"
    echo ""
}

# Stop backend
stop_backend() {
    if [ ! -f "$BACKEND_PID_FILE" ]; then
        warning "Backend PID file not found"
        return 1
    fi
    
    local pid=$(cat "$BACKEND_PID_FILE")
    
    if ! ps -p "$pid" > /dev/null 2>&1; then
        warning "Backend process not running"
        rm -f "$BACKEND_PID_FILE"
        return 1
    fi
    
    log "Stopping backend (PID: $pid)..."
    kill "$pid" || true
    
    sleep 2
    
    if ps -p "$pid" > /dev/null 2>&1; then
        warning "Forcing backend shutdown..."
        kill -9 "$pid" 2>/dev/null || true
    fi
    
    rm -f "$BACKEND_PID_FILE"
    success "Backend stopped"
}

# Stop frontend
stop_frontend() {
    if [ ! -f "$FRONTEND_PID_FILE" ]; then
        warning "Frontend PID file not found"
        return 1
    fi
    
    local pid=$(cat "$FRONTEND_PID_FILE")
    
    if ! ps -p "$pid" > /dev/null 2>&1; then
        warning "Frontend process not running"
        rm -f "$FRONTEND_PID_FILE"
        return 1
    fi
    
    log "Stopping frontend (PID: $pid)..."
    kill "$pid" || true
    
    sleep 1
    
    if ps -p "$pid" > /dev/null 2>&1; then
        warning "Forcing frontend shutdown..."
        kill -9 "$pid" 2>/dev/null || true
    fi
    
    rm -f "$FRONTEND_PID_FILE"
    success "Frontend stopped"
}

# Stop both
stop_all() {
    log "Stopping Axiom IDP..."
    echo ""
    
    stop_backend
    echo ""
    stop_frontend
    
    echo ""
    success "All services stopped"
}

# Status
status() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Axiom IDP Status${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    # Backend status
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            success "Backend:  RUNNING (PID: $pid, Port: 8080)"
            curl -s http://localhost:8080/health > /dev/null 2>&1 && \
                success "  Health check: RESPONDING" || \
                error "  Health check: NOT RESPONDING"
        else
            error "Backend:  NOT RUNNING (stale PID: $pid)"
        fi
    else
        error "Backend:  NOT RUNNING"
    fi
    
    echo ""
    
    # Frontend status
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            success "Frontend: RUNNING (PID: $pid, Port: 3000)"
            curl -s http://localhost:3000/ > /dev/null 2>&1 && \
                success "  Web server: RESPONDING" || \
                error "  Web server: NOT RESPONDING"
        else
            error "Frontend: NOT RUNNING (stale PID: $pid)"
        fi
    else
        error "Frontend: NOT RUNNING"
    fi
    
    echo ""
}

# Restart
restart() {
    log "Restarting Axiom IDP..."
    echo ""
    
    stop_all
    echo ""
    sleep 2
    echo ""
    start_all
}

# Tail logs
tail_logs() {
    log "Tailing application logs..."
    echo "Press Ctrl+C to exit"
    echo ""
    tail -f "$LOG_FILE"
}

# Main
case "$1" in
    start)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        tail_logs
        ;;
    *)
        echo "Axiom IDP Application Manager"
        echo ""
        echo "Usage: $0 {start|stop|restart|status|logs}"
        echo ""
        echo "Commands:"
        echo "  start    - Start backend and frontend servers"
        echo "  stop     - Stop all servers"
        echo "  restart  - Restart all servers"
        echo "  status   - Show status of all services"
        echo "  logs     - Tail application logs"
        echo ""
        exit 1
        ;;
esac
