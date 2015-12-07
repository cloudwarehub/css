package cache

import (
	"fmt"
    "github.com/cloudwarehub/css/ufile"
    "github.com/cloudwarehub/css/redis"
	"github.com/garyburd/redigo/redis"
)


type Cache struct{
    FileRedis myredis.MyRedis
    Context ufile.Context
}

func (cache *Cache) Init(host_port string, 	privateKey string, publicKey string, bucket string){
	c, err := redis.Dial("tcp", host_port)
	myredis := myredis.MyRedis{c}
	if err != nil {
		fmt.Println(err)
		return
	}else{
		fmt.Println("conncet success")
	}
	context := ufile.Context{
		PublicKey : publicKey,
		PrivateKey : privateKey,
		Bucket : bucket}
	cache.FileRedis = myredis
	cache.Context = context
}

func (cache *Cache) Get(file_id string) (interface{}, error) {
    v, err := cache.FileRedis.GetValue(file_id)
    if v == nil || err != nil{
        //从ufile里获取，然后存入缓存
		data, err := cache.Context.Get(file_id)
        if err != nil {
            return nil, err
        }else {
            cache.FileRedis.SetValue(file_id, data)
			return data, nil
        }
    }
    return v, nil
}

func (cache *Cache) Set(file_id string, data []byte) ([]byte, error) {
	cache.FileRedis.SetValue(file_id, data)
	v, err := cache.Context.Put(file_id, data)
	if err != nil {
		return nil, err
	}
	return v, nil
}

/*
func main() {
	publicKey := "ucloud1135032732@qq.com14476426960001214118939"
	privateKey := "b04362de5f4a1d16cdc4c00a33141a52c395aa61"
	bucket := "zkdnfcf"
	host_port := "10.10.168.190:6379"
	cache := Cache{}
	cache.Init(host_port, privateKey, publicKey, bucket)
	data, err := cache.Get("test")	
	if err != nil {
		fmt.Println(err)
	}else{
		fmt.Println(data)
	}

	s1 := "testtesttest111122"
	b1 := []byte(s1)
	data1, err1 := cache.Set("test", b1)
	fmt.Println(data1, err1)
	defer cache.FileRedis.Conn.Close()
}*/
