# totallyguysproject
put to cmd/server/.env ur local pg database address (DATABASE_URL)  
run app from cmd/server, (go run .)  
go to http://localhost:8080/swagger/index.html for swagger documentation

report 2/4:
# Movie Playlist Social App (Backend)

## 1. Project Description
This backend project is part of a web application that allows users to search for movies, create and manage playlists, leave reviews, participate in discussions, and interact socially around movies. Users can follow each other, comment on reviews, and build their own collections of favorite or watch-later movies. Admins have additional privileges to manage content across the platform.

## 2. Technology Stack
- **Programming Language:** Go  
- **Web Framework:** Gin  
- **ORM:** GORM  
- **Database:** PostgreSQL  

## 3. User Roles and Use Cases

### Roles
- **User:** Can search movies, create and manage personal playlists, write reviews, comment on others' reviews, follow other users, and engage socially.  
- **Admin:** All user capabilities, plus the ability to edit movie details, update descriptions, and delete reviews or comments made by any user.
  
## 4. Database Schema
### Use Case Diagram
![Use Case Diagram](sql/databaseTotallyGuys.png)

## 5. API
The backend API endpoints and their documentation are defined in the `server.go` file located in the `internal/server` directory.


# totallyweb
place this repo on the same level as totallyguysproject
