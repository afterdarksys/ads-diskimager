# Diskimager Web UI Frontend

Modern React-based web interface for forensic disk imaging operations.

## Quick Start

```bash
# Install dependencies
npm install

# Start development server (with hot reload)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## Development Setup

### Prerequisites
- Node.js 18+ and npm
- Diskimager backend running on port 8080

### Start Backend
```bash
# In parent directory
cd ../..
go build -o diskimager
./diskimager web --port 8080
```

### Start Frontend
```bash
# In this directory
npm install
npm run dev
```

Access at: `http://localhost:3000`

## Architecture

### Tech Stack
- **React 18** - UI framework
- **TypeScript** - Type safety
- **TailwindCSS** - Utility-first styling
- **Zustand** - Lightweight state management
- **Recharts** - Speed visualization
- **Axios** - HTTP client
- **Lucide React** - Icon library
- **Vite** - Build tool and dev server

### Project Structure
```
src/
├── components/          # React components
│   ├── Dashboard.tsx    # Main dashboard view
│   ├── DiskSelector.tsx # Disk list component
│   ├── JobQueue.tsx     # Job management component
│   └── ProgressMonitor.tsx # Real-time progress monitor
├── store/              # State management
│   └── jobStore.ts     # Zustand store for jobs
├── types/              # TypeScript types
│   └── index.ts        # Type definitions
├── App.tsx             # Main app component
├── main.tsx            # React entry point
└── index.css           # Global styles
```

## Features

### Dashboard
- Real-time job queue
- Disk selector with health indicators
- Live progress monitoring
- Speed graphs
- Error display

### Components

#### DiskSelector
- Lists available disks
- Shows health status
- Displays size and model
- Auto-refresh capability

#### JobQueue
- Real-time job list
- Progress bars
- Speed indicators
- Delete/Cancel buttons
- Status badges

#### ProgressMonitor
- WebSocket connection for live updates
- Speed graph (last 20 data points)
- Detailed metrics
- Error logging

## API Integration

### REST API
```typescript
// List disks
GET /api/disks

// Create imaging job
POST /api/jobs/image
{
  "source_path": "/dev/disk1",
  "dest_path": "evidence.e01",
  "format": "E01",
  "compression": 6,
  "metadata": {
    "case_number": "CASE-2026-001",
    "examiner": "John Doe"
  }
}

// List jobs
GET /api/jobs

// Get job details
GET /api/jobs/:id

// Cancel/Delete job
DELETE /api/jobs/:id
```

### WebSocket
```typescript
// Connect to job progress
const ws = new WebSocket('ws://localhost:8080/api/ws/jobs/:id');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // data.type: 'progress', 'status', 'error'
  // data.percentage, data.speed, data.eta, etc.
};
```

## State Management

### Zustand Store
```typescript
import { useJobStore } from './store/jobStore';

function Component() {
  const { jobs, disks, selectedJob } = useJobStore();
  const { addJob, updateJob, removeJob } = useJobStore();

  // Use state...
}
```

## Styling

### TailwindCSS
Custom forensic color palette:
```javascript
colors: {
  forensic: {
    50: '#f0f9ff',   // Lightest blue
    100: '#e0f2fe',
    // ...
    900: '#0c4a6e',  // Darkest blue
  }
}
```

Usage:
```jsx
<div className="bg-forensic-800 text-white">
  <button className="bg-forensic-600 hover:bg-forensic-700">
    Action
  </button>
</div>
```

## Development

### Hot Reload
Vite provides instant hot module replacement:
- Edit any file
- Changes appear immediately
- No page reload needed

### API Proxy
Vite proxies `/api/*` to backend:
```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8080'
  }
}
```

### Type Checking
```bash
# Run TypeScript compiler
npm run build

# Linting
npm run lint
```

## Production Build

### Build Process
```bash
npm run build
```

Creates:
- `dist/` - Optimized static files
- Minified JavaScript
- Optimized CSS
- Asset hashing

### Deployment Options

1. **Serve from Go** (recommended):
   ```go
   // Embed in Go binary
   //go:embed dist/*
   var webFiles embed.FS

   router.StaticFS("/", http.FS(webFiles))
   ```

2. **Nginx**:
   ```nginx
   server {
     listen 80;
     location / {
       root /path/to/dist;
       try_files $uri $uri/ /index.html;
     }
     location /api {
       proxy_pass http://localhost:8080;
     }
   }
   ```

3. **Docker**:
   ```dockerfile
   FROM node:18 AS builder
   WORKDIR /app
   COPY package*.json ./
   RUN npm install
   COPY . .
   RUN npm run build

   FROM nginx:alpine
   COPY --from=builder /app/dist /usr/share/nginx/html
   ```

## Troubleshooting

### Frontend Can't Connect to API
**Issue**: API calls fail with CORS or network errors

**Solution**:
1. Verify backend is running: `curl http://localhost:8080/health`
2. Check vite.config.ts proxy settings
3. Ensure CORS is enabled in Go backend

### WebSocket Connection Fails
**Issue**: Real-time updates don't work

**Solution**:
1. Check WebSocket URL (ws:// not wss:// for local)
2. Verify port is correct (8080)
3. Check browser console for errors
4. Ensure firewall allows WebSocket connections

### Build Errors
**Issue**: `npm run build` fails

**Solution**:
```bash
# Clear cache
rm -rf node_modules package-lock.json
npm install
npm run build
```

## Performance

### Optimization Tips
1. **Code Splitting**: Vite does this automatically
2. **Lazy Loading**: Use React.lazy() for routes
3. **Memoization**: Use React.memo() for expensive components
4. **WebSocket**: Only one connection per job
5. **Polling Fallback**: Polls every 2s if WebSocket unavailable

### Bundle Size
- Main bundle: ~150KB (gzipped)
- Vendor bundle: ~200KB (gzipped)
- Total: ~350KB (gzipped)

## Contributing

### Code Style
- Use TypeScript strict mode
- Follow React hooks best practices
- Use functional components
- Prefer composition over inheritance

### Git Workflow
1. Create feature branch
2. Make changes
3. Test locally
4. Build production
5. Create pull request

## Resources

- [React Documentation](https://react.dev)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [TailwindCSS Docs](https://tailwindcss.com/docs)
- [Zustand Guide](https://docs.pmnd.rs/zustand)
- [Vite Guide](https://vitejs.dev/guide/)

## License

© 2026 AfterDark Systems
