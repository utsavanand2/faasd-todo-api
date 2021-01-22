## CRUD API running on stateful faasd function with mongodb as the datastore

```sh
# Get all todos
curl http://142.93.222.75:8080/function/faas-mongo/todos

# Get todo by an author
curl http://142.93.222.75:8080/function/faas-mongo/get\?author\=Utsav

# Add a todo
curl http://142.93.222.75:8080/function/faas-mongo/add -d '{"todo":"fix add handler","author":"Utsav"}'

# Update a todo by ID
curl -X PUT http://142.93.222.75:8080/function/faas-mongo/update -d '{"id":"600ade20664052986314e439","todo":"Be The Person Your Dog Thinks You Are","author":"Utsav"}'

# Delete a todo
curl -X DELETE http://142.93.222.75:8080/function/faas-mongo/delete\?id\=\8237482hu34hyui3yt9384
```
