package cache

import {
    "fmt"
    "github.com/garyburd/redigo/redis"
    "gihub.com/cloudwarehub/css/redis/myredis"
    "gihub.com/cloudwarehub/css/ufile/ufile"
    "time"
}


type Cache struct{
    fileRedis MyRedis
    Context Context
}

func (cache *Cache) InitRedis(host_port string) {
    c, err := redis.DialTimeOut("tcp", host_port, 1*time.Second, 1*time.Second))
    if err != nil {
    fmt.Println(err)
        return
    }else{
        myredis := MyRedis{c}
        cache.fileRdis = myredis
		fmt.Println("conncet success")
	}
}


func (cache *Cache) Get(file_id string) ([]byte, error) {
    v, err := Cache.fileRdis.GetValue(file_id)
    if err != nil {
        //从ufile里获取，然后存入缓存
        v, err = cache.Context.Get(file_id)
        if err != nil {
            return nil, err
        }else {
            cache.fileRdis.SetValue(file_id, v)
        }
    }
    return v, nil
}


