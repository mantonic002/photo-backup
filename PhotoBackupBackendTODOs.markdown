# TODOs for Photo Backup Backend

- Install MongoDB
- Create `photos` collection with indexes (lonlat, taken_at)
- Sanitize filenames and handle duplicates
- Validate image content type
- Stream file writing for large uploads
- Thumbnail generation(`github.com/disintegration/imaging`)
- Thumbnail path
- Search methods (by location, time, both)
- Validate search query parameters
- Support batch uploads
- Custom error types for common issues
- Structured logging (`zap`)
- Clean up failed uploads
- JWT authentication
