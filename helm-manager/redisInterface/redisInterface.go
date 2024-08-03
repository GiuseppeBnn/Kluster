package redisInterface

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

const redisHost = "192.168.1.40"
const redisPort = "30207"

var redisClient = redis.NewClient(&redis.Options{
	Addr:     redisHost + ":" + redisPort,
	Password: "", // no password
	DB:       0,  // default DB
})

// funzione che crea un set con chiave key e valori values
func InsertInSet(key string, value string) error {
	ctx := context.Background()
	_, err := redisClient.SAdd(ctx, key, value).Result()
	if err != nil {
		log.Println("Could not insert in set: ", err)
		return err
	}
	return nil
}
func GetAllSetFromKey(key string) ([]string, error) {
	ctx := context.Background()
	val, err := redisClient.SMembers(ctx, key).Result()
	if err != nil {
		log.Println("Could not get key: ", err)
		return nil, err
	}
	return val, nil
}
func GetNumberOfSetFromKey(key string) (int64, error) {
	ctx := context.Background()
	val, err := redisClient.SCard(ctx, key).Result()
	if err != nil {
		log.Println("Could not get key: ", err)
		return 0, err
	}
	return val, nil
}

func CheckPresence(key string) (bool, error) {
	ctx := context.Background()
	val, err := redisClient.Exists(ctx, key).Result()
	if err != nil {
		log.Println("Could not check presence: ", err)
		return false, err
	}
	return val == 1, nil
}
func GetKeyValue(key string) (string, error) {
	ctx := context.Background()
	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		log.Println("Could not get key: ", err)
		return "", err
	}
	return val, nil
}

func DeleteFromSet(cf string, rel string) error {
	ctx := context.Background()
	_, err := redisClient.SRem(ctx, "rel-"+cf, rel).Result()
	if err != nil {
		log.Println("Could not delete from set: ", err)
		return err
	}
	log.Println("Deleted from set")
	return nil
}
