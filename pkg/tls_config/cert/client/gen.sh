# Создание ключа и CSR для клиента
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -config ../server/san.cnf

# Подписание CSR корневым сертификатом
openssl x509 -req -in client.csr -CA ../server/ca.crt -CAkey ../server/ca.key -CAcreateserial -out client.crt -days 365 -sha256 -extfile ../server/san.cnf -extensions v3_req