# Photo Backup App

This is a personal project designed as a self-hosted alternative to Google Photos, for storing, managing, and retrieving photos. The application allows users to upload photos, generate thumbnails, store metadata in MongoDB, and search photos by geolocation. It uses Go for the backend, MongoDB for the database, and JWT for authentication.

## Features

- **Photo Upload**: Upload photos (up to 200 MB for now) with automatic thumbnail generation.
- **Metadata Extraction**: Extracts EXIF data (e.g. geolocation, timestamp) from photos.
- **MongoDB Storage**: Stores photo metadata in a MongoDB database.
- **Geolocation Search**: Search for photos within a specified bounding box (min latitude, max latitude, min longitude and max longitude).
- **Secure Access**: Uses cookie session authentication for secure endpoints.
- **Logging**: Comprehensive logging with Zap for debugging and monitoring.
- **Local Storage**: Stores uploaded photos and thumbnails in a local directory.

## Prerequisites

- **Go**: Version 1.16 or higher.
- **MongoDB**: A running MongoDB instance (local or cloud-based).
- **Dependencies**:
  - `github.com/joho/godotenv` for environment variable management.
  - `go.uber.org/zap` for logging.
  - `go.mongodb.org/mongo-driver` for MongoDB connectivity.
  - `github.com/disintegration/imaging` for thumbnail generation.
  - `github.com/rwcarlsen/goexif` for EXIF data extraction.

## Setup Instructions

### 1. Clone the Repository

```bash
git clone <repository-url>
cd photo-backup
```

### 2. Configure Environment Variables

Edit `.env` file with your MongoDB configuration:

```plaintext
MONGO_URI=mongodb://localhost:27017
MONGO_DB=photo_backup
MONGO_COLLECTION=photos
```

Create a `.env.secret` file for sensitive information:

```plaintext
SESSION_SECRET=<your-32byte-session-secret>
PW='<bcrypt-hashed-password>'
```

To generate a bcrypt-hashed password, you can use a tool like `bcrypt-cli` or an online bcrypt generator. Example using a Go bcrypt library:

```bash
package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "yourpassword"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	fmt.Println(string(hashed))
}
```

### 3. Set Up MongoDB

Make sure MongoDB is running and accessible via the `MONGO_URI` specified in `.env`. Create a database named `photo_backup` with a collection named `photos`. The following indexes are required:

- `_id` (default index, automatically created).
- `taken_at` (descending, for sorting by timestamp).
- `lonlat` (2dsphere, for geolocation queries).

Run the following MongoDB commands to create the indexes:

```javascript
use photo_backup
db.photos.createIndex({ "taken_at": -1 })
db.photos.createIndex({ "lonlat": "2dsphere" }, { sparse: true })
```

### 4. Install Dependencies

```bash
go mod tidy
```

### 5. Create Upload Directory

Create a directory for storing uploaded photos and thumbnails:

```bash
mkdir .uploads
```

### 6. Run the Application

```bash
go run main.go
```

The server will start on `http://localhost:8080`.

## API Endpoints

### Authentication

- **POST /login**
  - Authenticate using a password to receive a JWT token.
  - Body: `{"password": "<your-password>"}`
  - Response: `{"token": "<jwt-token>"}`

### Photo Management

- **GET /photos?id=<photo-id>**
  - Retrieve a single photo by ID (returns full-size photo metadata).
  - Secured.
- **GET /photos?lastId=<last-id>&limit=<limit>**
  - Retrieve a paginated list of photos (returns thumbnail metadata).
  - Secured.
- **POST /photos**
  - Upload one or more photos (multipart form with `file` field).
  - Secured.
  - Max file size: 200 MB.
- **GET /photos/search?latMin=&latMax=&longMin=&longMax=**
  - Search photos by geolocation within a specified bounding box.
  - Secured.
- **GET /files/<filename>**
  - Serve a photo or thumbnail file from the `.uploads` directory.
  - Secured.

## Project Structure

- `main.go`: Entry point, initializes the server, MongoDB, and routes.
- `api/photo_handlers.go`: Handles HTTP requests and routes for photo operations.-
  `api/auth.go`: contains `handleLogin` handler function.
- `storage/photo_storage.go`: Manages local file storage and thumbnail generation.
- `storage/photo_db.go`: Interacts with MongoDB for photo metadata.
- `model/photo.go`: Contains data models (`PhotoDB`, `GeoPoint`).

## Usage Example

1. **Login**:

   ```bash
   curl -X POST http://localhost:8080/login -d '{"password": "<your-password>"}'
   ```

   Copy the returned JWT token.

2. **Upload a Photo**:

   ```bash
   curl -X POST http://localhost:8080/photos  -F "file=@/path/to/photo.jpg"
   ```

3. **Retrieve Photos**:

   ```bash
   curl "http://localhost:8080/photos?lastId=&limit=10"
   ```

4. **Search Photos by Location**:

   ```bash
   curl "http://localhost:8080/photos/search?latMin=0&latMax=1&longMin=0&longMax=1"
   ```

5. **Retrieve a Served File**:
   To retrieve a full-size photo or thumbnail from the .Uploads directory, use the `/files/<filename>` endpoint. The filename can be obtained from the FilePath or ThumbnailPath fields in the response from the `/photos` or `/photos/search` endpoints.

   ```bash
   curl "http://localhost:8080/files/<photo-id>.jpg"  -o photo.jpg
   ```

   Example with a specific photo ID (e.g. `1234567890abcdef12345678`):

   ```bash
   curl "http://localhost:8080/files/1234567890abcdef12345678.jpg"  -o photo.jpg
   ```

   To retrieve a thumbnail:

   ```bash
   curl "http://localhost:8080/files/1234567890abcdef12345678_thumb.jpg"  -o thumbnail.jpg
   ```

## Notes

- Ensure the `.uploads` directory has sufficient storage space.
- The application generates thumbnails (100x100 pixels) using the `imaging` library.
- EXIF data is extracted for geolocation and timestamp; if unavailable, defaults are used.
- All endpoints except `/login` require a valid session cookie set.

## Contributing

This is a personal project, but feel free to fork and modify it for your needs. Suggestions or improvements can be submitted via pull requests.
