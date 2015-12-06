package cache

import {
    "fmt"
    "github.com/garyburd/redigo/redis"
}


type Cache struct{
}

func (cache *Cache) Get(file_id string) (byte[], error) {
    
}