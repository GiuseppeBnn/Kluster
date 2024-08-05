const redis = require('redis');

class redisInterface {
    constructor(host, port) {
        this.redis_client = null;
        this.setClient(host, port);
    }

    setClient(host, port) {
        this.redis_client = redis.createClient({url: `redis://${host}:${port}`});
        console.log(`Redis client created at ${host}:${port}`);
        this.redis_client.on('error', function(err) {
            console.log(err);
            return this.redis_client.quit();
        });
    }
    async setKeyValue(key, value, expiry) {
        await this.redis_client.connect();
        if(expiry){
            this.redis_client.set(key, value, "EX", expiry)
        }
        else {
            this.redis_client.set(key, value);
        }
        await this.redis_client.quit();
    }

    async readValueFromKey(key) {
        await this.redis_client.connect();
        let val=await this.redis_client.get(key);
        await this.redis_client.quit();
        return val;
    }

    async checkTokenPresence(token) {
        await this.redis_client.connect();
        let b=await this.redis_client.exists(token); 
        await this.redis_client.quit();
        return b;       
    }

    async delKey(key) {
        await this.redis_client.connect();
        await this.redis_client.del(key);
        await this.redis_client.quit();
    }
}





module.exports = redisInterface;