# 📝 Todo List Application

A simple yet powerful todo list application built with Go. Create todos, mark them as complete, and organize your tasks efficiently.

## Current Features

- Simple todo list management
- Add todos with a checkbox to mark them complete
- Cross off completed todos
- REST API backend built with Go
- Basic HTTP server on port 8080

## Getting Started

### Prerequisites
- Go 1.16 or higher
- Git

### Installation & Running

1. Clone:
```bash
git clone https://github.com/snehachetani/ToDo-List.git
```

2. Run the application:
```bash
go run web-api.go
```

3. The server will start on `http://localhost:8080`

### API Endpoints

#### Home
```
GET /
```
Returns a welcome message.

#### Show Tasks
```
GET /show-tasks
```
Returns all current tasks in the list.

**Example Response:**
```
Watch Go crash course
Watch Nana's Golang Full Course
Reward myself with a donut
```

## 📋 Project Roadmap

### Phase 1: Core API Features (Next)
- [ ] `POST /tasks` - Add a new task
- [ ] `PUT /tasks/:id` - Mark task as complete/incomplete
- [ ] `DELETE /tasks/:id` - Delete a task
- [ ] `GET /tasks/:id` - Get a specific task
- [ ] Task persistence (in-memory or file-based)

### Phase 2: Data Persistence
- [ ] Connect to a database (PostgreSQL or MongoDB)
- [ ] Implement task model with fields: `id`, `title`, `description`, `completed`, `createdAt`, `dueDate`
- [ ] Add database migrations

### Phase 3: User Authentication
- [ ] User registration and login
- [ ] JWT token authentication
- [ ] Secure endpoints with authentication middleware
- [ ] User-specific task lists

### Phase 4: Enhanced Features
- [ ] Task categories/projects
- [ ] Priority levels (High, Medium, Low)
- [ ] Due dates and reminders
- [ ] Task descriptions and notes
- [ ] Search and filter functionality
- [ ] Sort by priority, due date, or creation date

### Phase 5: Frontend
- [ ] Web UI (React, Vue, or vanilla HTML/CSS/JS)
- [ ] Interactive task management interface
- [ ] Real-time updates with WebSockets
- [ ] Mobile-responsive design
- [ ] Dark mode support

### Phase 6: Advanced Features
- [ ] Task sharing and collaboration
- [ ] Recurring tasks
- [ ] Task attachments
- [ ] Notifications and email reminders
- [ ] Analytics dashboard
- [ ] Undo/Redo functionality

### Phase 7: DevOps & Deployment
- [ ] Docker containerization
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Unit and integration tests
- [ ] API documentation (Swagger/OpenAPI)
- [ ] Deploy to cloud (AWS, Heroku, DigitalOcean)

### Phase 8: Polish & Scale
- [ ] Performance optimization
- [ ] Rate limiting
- [ ] Logging and monitoring
- [ ] Error handling improvements
- [ ] API versioning


---

**Happy organizing! 🎉**
