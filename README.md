# Kartex Image Provider Service

A high-performance image handling service that provides efficient storage, retrieval, and conversion of images. The service automatically converts all images to WebP format for optimal performance and storage efficiency.
This service is designed for publicly serving files from securely accessed services like S3. Regardless of the original image format stored in secure S3, it always delivers a compressed WebP version, with memory first efficiency as the top priority.


## Features

- **Multi-Storage Support**
  - Primary storage: Local filesystem
  - Secondary storage: S3/MinIO (optional)
  - Automatic fallback to secondary storage when files are not found locally

- **Smart Caching**
  - In-memory cache for the last default 1000 (configured) accessed images
  - FIFO (First-In-First-Out) cache eviction policy
  - Automatic cache population from storage

- **Efficient File Organization**
  - Files stored in a hierarchical directory structure based on GUID
  - Example: Image ID "123456" is stored as "12/34/56.webp"
  - All files stored in WebP format for optimal performance

- **Security Features**
  - API key authentication
  - Rate limiting
  - CORS protection
  - Secure server configuration

## API Endpoints

### Public Routes
- `GET /health` - Health check endpoint
- `GET /images/:id` - Get an image by ID

### Protected Routes (Requires API Key)
- `POST /images` - Upload a new image
- `DELETE /images/:id` - Delete an image
- `GET /images` - List all images

## Configuration

Create a `.env` file with the following settings:

```env
# Server Configuration
BIND_ADDRESS=:8080  # Server bind address (e.g., :8080, 0.0.0.0:8080, localhost:8080)

# Storage Configuration
STORAGE_TYPE=local
STORAGE_PATH=./data  # Directory where files will be stored

# S3/MinIO Configuration (Optional)
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin
AWS_REGION=us-east-1
AWS_BUCKET_NAME=images
MINIO_ENDPOINT=http://localhost:9000
MINIO_USE_SSL=false

# Security
API_KEY=your_api_key_here
RATE_LIMIT=100
RATE_LIMIT_WINDOW=60
ALLOWED_ORIGINS=http://localhost:3000,https://yourdomain.com
```

## File Storage Structure

The service uses a hierarchical directory structure for efficient file organization:

```
data/
├── 12/
│   └── 34/
│       └── 56.webp
├── ab/
│   └── cd/
│       └── ef.webp
└── ...
```

- Each image is stored with its GUID as the filename
- The GUID is split into parts to create the directory structure
- All files are stored in WebP format
- When retrieving from S3/MinIO, images are automatically converted to WebP

## Image Handling

- All images are automatically converted to WebP format
- When requesting an image (e.g., `xmas.jpg`), the service will:
  1. Check cache for any version of the file
  2. If found in WebP format, return it directly
  3. If found in another format, convert to WebP and return
  4. If not in cache, check storage and convert to WebP if needed

## Getting Started

1. Clone the repository
2. Create a `.env` file with your configuration
3. Build and run the service:

```bash
go build -o imageprovider cmd/api/main.go
./imageprovider
```

## API Usage Examples

### Upload an Image
```bash
curl -X POST http://localhost:8080/images \
  -H "X-API-Key: your_api_key" \
  -F "file=@/path/to/image.jpg"
```

### Get an Image
```bash
curl -O http://localhost:8080/images/123456
```

### Delete an Image
```bash
curl -X DELETE http://localhost:8080/images/123456 \
  -H "X-API-Key: your_api_key"
```

### List All Images
```bash
curl http://localhost:8080/images \
  -H "X-API-Key: your_api_key"
```

## Error Handling

The service provides clear error messages for common scenarios:
- 401: Invalid or missing API key
- 404: Image not found
- 429: Rate limit exceeded
- 500: Internal server error

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 