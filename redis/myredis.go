package myredis

import (
    "fmt"
    "github.com/garyburd/redigo/redis"
)

type MyRedis struct{
    Conn redis.Conn
}


func (red *MyRedis) GetValue(file_id string) (interface{}, error) {
    file_id = "123"
	v, err := red.Conn.Do("GET", file_id)
    if err != nil {
    	fmt.Println(err)
        return nil, err
    }
	return v, nil
}

func (red *MyRedis) SetValue(file_id string, data []byte) (interface{}, error) {
    v, err := red.Conn.Do("SET", file_id, data)
    return v, err
}

/*
func main() {
	// for test
    c, err := redis.Dial("tcp", "10.10.168.190:6379")
    conn := MyRedis{c}
    if err != nil {
    fmt.Println(err)
        return
    }else{
		fmt.Println("conncet success")
	}
	str := "123"
    b1 := []byte(str)
 	conn.SetValue("zkdnfcf1", b1)

    data, err := conn.GetValue("zkdnfcf1")
    if err != nil {
        fmt.Println(err)
        return
    }else {
        fmt.Println(data)
    }

    defer c.Close()
}*/
