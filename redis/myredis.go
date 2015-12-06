package redis

import {
    "fmt"
    "github.com/garyburd/redigo/redis"
}

type MyRedis struct{
    rconn *redis.conn
}

func (redis *MyRedis) GetValue(file_id string) ([]byte, err) {
    v, err = redis.String(c.Do("GET", string))
    if err != nil {
    fmt.Println(err)
        return
    }
    fmt.Println(v)
}

func (redis *MyRedis) SetValue(file_id string, data []byte) (err) {
    v, err := redis.rconn.Do("SET", string, data)
    return err
}

func main() {
    c, err := redis.Dial("tcp", "10.10.168.190:6379")
    conn := MyRedis(c)
    if err != nil {
    fmt.Println(err)
        return
    }
    
    str := "123"
    b1 := []byte(str)
    err := conn.SetValue("zkdnfcf1", b1)
    if err != nil {
        fmt.Println(err)
        return
    }
    
    data, err := conn.GetValue("zkdnfcf1")
    if err != nil {
        fmt.Println(err)
        return
    }else {
        fmt.Println(data)
    }
    defer c.Close()
}