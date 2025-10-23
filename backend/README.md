### How to setup 

## To access Postgres in docker
Notes: \dt, \d media, use "\x on" for readable sql query result
```bash
docker exec -it pg psql -U postgres -d media
```

## Initiate Postgres
Prereq: install goose migration utility 
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```
- Add goose to your PATH (if not done yet): 
```bash
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
source ~/.zshrc
```
Apply migration on running Postgres in Docker: 
```bash
goose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/media?sslmode=disable" up
```

### Test upload route 
```bash
curl -X POST http://localhost:8080/api/upload/image \
  -F "files[]=@/home/nero/testdata/pic1.jpg" \
  -F "files[]=@/home/nero/testdata/pic2.jpg" \
  -F "files[]=@/home/nero/testdata/pic3.jpg" \
  -v
```

```bash
curl -X POST http://localhost:8080/api/upload/image \
  -F "files[]=@/Users/fredddhdjd/prj/AI_photo_retrieval_system/backend/test/pic1.jpg" \
  -F "files[]=@/Users/fredddhdjd/prj/AI_photo_retrieval_system/backend/test/pic2.jpg" \
  -F "files[]=@/Users/fredddhdjd/prj/AI_photo_retrieval_system/backend/test/pic3.png" \
  -v
```

### Search route
```bash
curl -X POST "http://localhost:8080/api/search/text" \
  -H "Content-Type: application/json" \
  -d '{"user_id":404, "q":"cat pictures"}'
```
