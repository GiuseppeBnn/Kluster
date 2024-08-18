const redis = require("redis");

class redisInterface {
  constructor(host, port) {
    this.host = host;
    this.port = port;
  }

  getNewClient() {
    const redis_client = redis.createClient({
      url: `redis://${this.host}:${this.port}`,
    });
    redis_client.on("error", function (err) {
      console.log(err);
      return redis_client.quit();
    });
    return redis_client;
  }
  async setKeyValue(key, value, expiry) {
    const redis_client = this.getNewClient();
    await redis_client.connect();
    if (expiry) {
      redis_client.set(key, value, "EX", expiry);
    } else {
      redis_client.set(key, value);
    }
    redis_client.quit();
  }

  async readValueFromKey(key) {
    try {
      const redis_client = this.getNewClient();
      await redis_client.connect();
      let val = await redis_client.get(key);
      await redis_client.quit();
      return val;
    } catch (err) {
      console.error(err);
      redis_client.quit();
      return null;
    }
  }

  async checkTokenPresence(token) {
    try {
      const redis_client = this.getNewClient();
      await redis_client.connect();
      let b = await redis_client.exists(token);
      await redis_client.quit();
      return b;
    } catch (err) {
      console.error("Token verification failed:", err);
      return null;
    }
  }

  async delKey(key) {
    const redis_client = this.getNewClient();
    await redis_client.connect();
    await redis_client.del(key);
    await redis_client.quit();
  }
}

module.exports = redisInterface;
