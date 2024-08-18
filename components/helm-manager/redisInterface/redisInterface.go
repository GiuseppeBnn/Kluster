package redisInterface

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

const redisHost = "redis"
const redisPort = "6379"

func getNewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: "", // no password
		DB:       0,  // default DB
	})
}

// funzione che crea un set con chiave key e valori values
func InsertInSet(key string, value string) error {
	ctx := context.Background()
	redisClient := getNewRedisClient()
	_, err := redisClient.SAdd(ctx, key, value).Result()
	if err != nil {
		log.Println("(InsertInSet)Could not insert in set: ", err)
		return err
	}
	return nil
}
func GetAllSetFromKey(key string) ([]string, error) {
	ctx := context.Background()
	redisClient := getNewRedisClient()
	val, err := redisClient.SMembers(ctx, key).Result()
	if err != nil {
		log.Println("(GetAllSetFromKey)Could not get key: ", err)
		return nil, err
	}
	return val, nil
}
func GetNumberOfSetFromKey(key string) (int64, error) {
	ctx := context.Background()
	redisClient := getNewRedisClient()
	val, err := redisClient.SCard(ctx, key).Result()
	if err != nil {
		log.Println("(GetNumberOfSetFromKey)Could not get key: ", err)
		return 0, err
	}
	return val, nil
}

func CheckPresence(key string) (bool, error) {
	ctx := context.Background()
	redisClient := getNewRedisClient()
	val, err := redisClient.Exists(ctx, key).Result()
	if err != nil {
		log.Println("(CheckPresence)Could not check presence: ", err)
		return false, err
	}
	return val == 1, nil
}
func GetKeyValue(key string) (string, error) {
	ctx := context.Background()
	redisClient := getNewRedisClient()
	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		log.Println("(GetKeyValue)Could not get key: ", err)
		return "", err
	}
	return val, nil
}

func DeleteFromSet(cf string, rel string) error {
	ctx := context.Background()
	redisClient := getNewRedisClient()
	_, err := redisClient.SRem(ctx, "rel-"+cf, rel).Result()
	if err != nil {
		log.Println("(DeleteFromSet)Could not delete from set: ", err)
		return err
	}
	return nil
}
