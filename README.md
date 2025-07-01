# real-time-forum-typing-in-progress

A real-time web forum application that enables users to communicate, create posts, comment, like/dislike, filter posts, and chat privately with other users. The project features a modern frontend, a Go backend, and real-time communication via WebSockets.

## Features

- **User Authentication**: Register and log in with nickname/email, password, and profile details.
- **Real-Time Chat**: Private messaging between users with typing indicators and unread message notifications.
- **Posts & Comments**: Create, view, like/dislike, and comment on posts.
- **Categories**: Organize posts by categories and filter them.
- **Notifications**: Real-time notifications for new messages and interactions.
- **Session Management**: Secure session handling with middleware.
- **Responsive Frontend**: Modern UI built with HTML, CSS, and JavaScript.

## Project Structure

```
forum.db
go.mod
main.go
backend/
  auth/
  categories/
  chat/
  comments/
  interactions/
  middleware/
  models/
  posts/
  utils/
database/
frontend/
  assets/
  css/
  html/
  js/
```

- **backend/**: Go backend logic (authentication, chat, posts, etc.)
- **database/**: Database connection and setup
- **frontend/**: Static assets and client-side code

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js (for frontend development, optional)
- SQLite3 (or use the provided `forum.db`)

### Installation

1. **Clone the repository**
   ```sh
   git clone https://github.com/Ouachanii/real-time-forum-typing-in-progress.git
   cd real-time-forum
   ```

2. **Install Go dependencies**
   ```sh
   go mod download
   ```

3. **Run the server**
   ```sh
   go run main.go
   ```
   The server will start at [http://localhost:3344/](http://localhost:3344/).

4. **Access the frontend**
   Open your browser and navigate to [http://localhost:3344/frontend/html/index.html](http://localhost:3344/frontend/html/index.html)

## Usage

- Register a new account or log in.
- View and create posts, filter by categories.
- Like/dislike posts and comments.
- Chat privately with other users in real time.
- Receive notifications for new messages and interactions.

## Technologies Used

- **Backend**: Go, Gorilla WebSocket, SQLite3
- **Frontend**: HTML, CSS, JavaScript
- **Database**: SQLite

## Development

- Backend entry point: [`main.go`](main.go)
- Frontend entry point: [`frontend/html/index.html`](frontend/html/index.html)
- Real-time chat logic: [`frontend/js/chat.js`](frontend/js/chat.js), [`backend/chat/`](backend/chat/)
- Authentication: [`backend/auth/`](backend/auth/), [`frontend/js/auth.js`](frontend/js/auth.js)

## License

Apache License 2.0

Copyright 2025 abouachani, asoudri

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Authors

asoudri && abouachani

---

*This project is for educational purposes and demonstrates a full-stack real-time web application in Go.*
