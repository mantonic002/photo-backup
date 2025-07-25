# TODOs for Photo Backup Backend

- Install PostgreSQL and PostGIS
- Create database and enable PostGIS
- Create `photos` table with indexes (lonlat, taken_at, metadata)
- Sstore metadata in PostgreSQL
- Sanitize filenames and handle duplicates
- Validate image content type
- Stream file writing for large uploads
- Thumbnail generation(`github.com/disintegration/imaging`)
- Thumbnail path in `photos` table (`thumbnail_path` column)
- Search methods (by location, time, both)
- Validate search query parameters
- Support batch uploads
- Custom error types for common issues
- Structured logging (`zap`)
- Clean up failed uploads
- JWT authentication
