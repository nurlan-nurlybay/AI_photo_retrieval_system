### How to setup 

## Apply migration
Start postgres in docker and use goose migration
```bash
goose -dir ./backend/migrations postgres "postgres://postgres:postgres@localhost:5432/media?sslmode=disable" up
```

## Init DB
Prereq: ensure that u have postgres running and create ur actuall db 
install goose: 
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```
- Add to your PATH (if not done yet): 
```bash
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
source ~/.zshrc
```
apply migration: 
```bash
goose -dir ./migrations postgres "postgres://postgres:postgres@172.22.16.1:5432/media?sslmode=disable" up
```

### Test upload route 

```bash
curl -X POST http://localhost:8080/api/upload/image \
  -F "files[]=@/home/nero/testdata/pic1.jpg" \
  -F "files[]=@/home/nero/testdata/pic2.jpg" \
  -F "files[]=@/home/nero/testdata/pic3.jpg" \
  -v
```

### Search route
```bash
curl -X POST "http://localhost:8080/api/search/text" \
  -H "Content-Type: application/json" \
  -d '{"user_id":404, "q":"cat pictures"}'
```
