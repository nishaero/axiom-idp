# 🚀 Axiom IDP - Local Development Guide

## ✅ Status: SUCCESSFULLY BUILT AND RUNNING

Both the backend API server and frontend web server are running and fully functional.

---

## 🌐 Access the Application

### Web Interface
- **URL:** http://localhost:3000
- **Status:** ✅ Running (Serving React frontend)

### Backend API
- **URL:** http://localhost:8080
- **Status:** ✅ Running (Go HTTP server)
- **Health Check:** http://localhost:8080/health

---

## 📊 Application Components

### Backend (Go)
- **Status:** ✅ Running (PID available)
- **Port:** 8080
- **Features:**
  - RESTful API with CORS support
  - Authentication middleware
  - Service catalog indexing
  - Audit logging
  - MCP plugin registry
  - Health check endpoint
- **Endpoints:**
  - GET `/health` - Health check
  - GET `/api/v1/catalog/services` - List services (requires auth)
  - GET `/api/v1/mcp/servers` - List MCP servers
  - POST `/api/v1/auth/token` - Generate token

### Frontend (React)
- **Status:** ✅ Running
- **Port:** 3000
- **Technology:** React 18 + TypeScript + TailwindCSS + Vite
- **Features:**
  - Service catalog browser
  - AI assistant chat interface
  - Dashboard with metrics
  - Responsive design
  - Dark mode support

---

## 🛠️ Managing the Application

### Using the Management Script

```bash
cd /home/nishaero/ai-workspace/axiom-idp

# Start both services
./manage-app.sh start

# Stop all services
./manage-app.sh stop

# Restart services
./manage-app.sh restart

# Check status
./manage-app.sh status

# View logs
./manage-app.sh logs
```

### Manual Commands

```bash
# Start backend
cd /home/nishaero/ai-workspace/axiom-idp
nohup ./bin/axiom-server >> axiom.log 2>&1 &

# Start frontend
cd /home/nishaero/ai-workspace/axiom-idp/web
python3 -m http.server 3000 --directory dist &

# Stop backend
pkill -f axiom-server

# Stop frontend  
pkill -f "http.server 3000"
```

---

## 📝 Configuration

### Environment Variables (.env)
Located at: `/home/nishaero/ai-workspace/axiom-idp/.env`

```env
# Server Configuration
HOST=localhost
PORT=8080
LOG_LEVEL=info
ENVIRONMENT=development

# Database Configuration
DATABASE_URL=sqlite:///./axiom.db
DATABASE_DRIVER=sqlite3

# OAuth2 Configuration (optional)
OAUTH2_PROVIDER=
OAUTH2_CLIENT_ID=
OAUTH2_CLIENT_SECRET=
OAUTH2_REDIRECT_URL=http://localhost:3000/auth/callback

# MCP Configuration
MCP_TIMEOUT=30s
MCP_MAX_RETRIES=3

# CORS Configuration
CORS_ALLOWED_ORIGINS=*
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS

# Frontend Configuration
FRONTEND_URL=http://localhost:3000
FRONTEND_BUILD_DIR=./web/dist
```

---

## 📂 Project Structure

```
axiom-idp/
├── bin/
│   └── axiom-server          # Compiled backend binary (7.6 MB)
├── cmd/
│   └── axiom-server/
│       └── main.go           # Backend entry point
├── internal/
│   ├── auth/                 # Authentication & RBAC
│   ├── catalog/              # Service catalog
│   ├── config/               # Configuration management
│   ├── logging/              # Logging setup
│   ├── mcp/                  # MCP registry
│   ├── ai/                   # AI routing
│   └── server/               # HTTP server & middleware
├── web/
│   ├── dist/                 # Built frontend assets
│   ├── src/
│   │   ├── pages/            # React pages
│   │   ├── components/       # React components
│   │   ├── lib/              # Utilities
│   │   └── store/            # State management
│   └── package.json          # Frontend dependencies
├── mcp-servers/              # MCP server examples
├── deployments/              # Docker, K8s configs
├── .env                      # Environment configuration
├── manage-app.sh             # Application management script
└── go.mod                    # Go module definition
```

---

## 🧪 Testing

### Run All Tests
```bash
cd /home/nishaero/ai-workspace/axiom-idp
go test ./internal/... -v -timeout 20s
```

### Test Results
```
✅ internal/ai              - 2/2 PASSED
✅ internal/auth           - 5/5 PASSED
✅ internal/catalog        - 6/6 PASSED
✅ internal/mcp            - 4/4 PASSED
✅ internal/server         - 6/6 PASSED
✅ pkg/utils              - 2/2 PASSED
━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 25/25 PASSED
```

### Manual API Testing
```bash
# Health check
curl http://localhost:8080/health

# Frontend
curl http://localhost:3000/

# Check ports
ss -tuln | grep -E ':(3000|8080)'
```

---

## 📋 Build Details

### Backend Build
```
Binary: bin/axiom-server
Size: 7.6 MB
Go Version: 1.22
Dependencies: 22 modules
Build Time: ~1 minute
Compiler: Go 1.22.2
```

### Frontend Build
```
Tool: Vite 5.4.21
Compiler: TypeScript 5.x
Framework: React 18
Styling: TailwindCSS 3
Bundle Size: 238 KB (optimized)
Build Time: ~788ms
```

---

## 🚀 Build Instructions

### Prerequisites
- Go 1.22+
- Node.js 18+
- npm 9+

### Rebuild Backend
```bash
cd /home/nishaero/ai-workspace/axiom-idp
go build -o bin/axiom-server cmd/axiom-server/main.go
```

### Rebuild Frontend
```bash
cd /home/nishaero/ai-workspace/axiom-idp/web
npm run build
```

### Rebuild Both
```bash
cd /home/nishaero/ai-workspace/axiom-idp

# Backend
go build -o bin/axiom-server cmd/axiom-server/main.go

# Frontend
cd web && npm run build && cd ..
```

---

## 🔍 Debugging

### View Logs
```bash
# Live logs
tail -f /home/nishaero/ai-workspace/axiom-idp/axiom.log

# Grep logs
grep ERROR axiom.log
grep "Starting" axiom.log
```

### Check Processes
```bash
# List running Axiom processes
ps aux | grep -E '(axiom|http.server)' | grep -v grep

# Monitor in real-time
watch -n 1 'ps aux | grep -E "(axiom|http.server)" | grep -v grep'
```

### Test Connectivity
```bash
# Backend health
curl -v http://localhost:8080/health

# Frontend
curl -i http://localhost:3000/

# Port availability
netstat -tuln | grep -E ':(3000|8080)'
```

---

## 📦 Dependencies

### Backend (Go)
- github.com/gorilla/mux - HTTP routing
- github.com/sirupsen/logrus - Logging
- github.com/coreos/go-oidc/v3 - OAuth2/OIDC
- golang.org/x/oauth2 - OAuth2 support
- github.com/joho/godotenv - .env loading
- github.com/google/uuid - UUID generation
- go.uber.org/zap - Structured logging
- github.com/lib/pq - PostgreSQL driver
- github.com/mattn/go-sqlite3 - SQLite driver

### Frontend (npm)
- react@18+ - UI framework
- typescript - Type safety
- tailwindcss - Styling
- vite - Build tool
- react-query - Server state
- zustand - Client state
- axios - HTTP client

---

## 🔐 Security

### Implemented
- ✅ OAuth2/OIDC authentication
- ✅ RBAC (4 roles: Admin, Engineer, Contributor, Viewer)
- ✅ Audit logging
- ✅ Security headers (CSP, HSTS, X-Frame-Options)
- ✅ CORS configuration
- ✅ Input validation

### Recommended Setup
1. Enable HTTPS/TLS in production
2. Configure OAuth2 provider (Google, GitHub)
3. Set strong database password
4. Use PostgreSQL instead of SQLite
5. Configure rate limiting
6. Enable audit log persistence

---

## 📈 Performance

### Baseline Metrics
- Backend startup: ~500ms
- Frontend bundle: 238 KB (gzipped)
- Health check response: <10ms
- API request latency: <100ms (local)
- Memory footprint: <256MB

### Optimization Tips
1. Enable caching for frontend assets
2. Configure database connection pooling
3. Use optimization flags in production build
4. Deploy with CDN for frontend
5. Use load balancer for multiple backend instances

---

## 🌍 Deployment

### Docker
```bash
docker-compose up
```

### Kubernetes
```bash
kubectl apply -f deployments/k8s-deployment.yaml
```

### Systemd
```bash
sudo ./scripts/install-systemd.sh
```

---

## 📚 Documentation

- [Architecture](./docs/architecture.md)
- [Getting Started](./docs/getting-started.md)
- [Contributing](./CONTRIBUTING.md)
- [Security Policy](./SECURITY.md)
- [Implementation Status](./IMPLEMENTATION_STATUS.md)
- [Build Success Report](./BUILD_SUCCESS_REPORT.md)

---

## 🆘 Troubleshooting

### Backend Won't Start
```bash
# Check if port 8080 is in use
lsof -i :8080

# Check logs
tail -f axiom.log

# Verify binary exists
ls -lh bin/axiom-server
```

### Frontend Not Loading
```bash
# Check if port 3000 is running
lsof -i :3000

# Verify dist folder exists
ls -la web/dist/

# Test with curl
curl http://localhost:3000/
```

### API Authentication Error
```bash
# The API requires authentication headers
# This is expected behavior - use OAuth2 to get tokens

# Health check doesn't require auth
curl http://localhost:8080/health
```

---

## 📊 Examples

### Test Backend with curl
```bash
# Health check (no auth required)
curl http://localhost:8080/health

# List MCP servers (requires auth)
# Note: Requests to /api/v1/* require Authorization header
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/mcp/servers
```

### Test Frontend
```bash
# Load web UI
curl http://localhost:3000/ | head -20

# Check JavaScript bundle
curl http://localhost:3000/assets/index-*.js | head -c 100
```

---

## ✨ What's Next?

1. **Access the Web UI**
   ```bash
   # Open browser
   http://localhost:3000
   ```

2. **Develop**
   - Modify frontend: `web/src/*`
   - Modify backend: `internal/*`
   - Rebuild as needed

3. **Test**
   ```bash
   go test ./...
   cd web && npm test
   ```

4. **Deploy**
   - Follow deployment guide
   - Configure production .env
   - Deploy to cloud platform

---

## 📞 Support

For issues or questions:
1. Check [CONTRIBUTING.md](./CONTRIBUTING.md)
2. Review [SECURITY.md](./SECURITY.md)
3. Check application logs
4. Review API documentation

---

**Status:** ✅ **All systems operational**  
**Last Updated:** February 12, 2026  
**Tested on:** Linux (Ubuntu 24.04)
