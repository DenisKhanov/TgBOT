# Создание корневого сертификата
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1024 -out ca.crt -subj "/CN=MyRootCA" -config san.cnf


# Создание ключа и CSR для сервера
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -config san.cnf

# Подписание CSR корневым сертификатом
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256 -extfile san.cnf -extensions v3_req
