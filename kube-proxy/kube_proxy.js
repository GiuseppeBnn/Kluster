const express = require("express");
const RedisInterface = require("./redisInterface");
const proxy = require("express-http-proxy");
const redisInterface = new RedisInterface("redis", 6379); //da cambiare con ip e porta corretti

const app = express();
const proxyServerPort = 3001;

const proxyCache = new Map();

async function checkTokenPresence(token) {
  let b = await redisInterface.checkTokenPresence(token);
  return b;
}
async function getValuesFromToken(token) {
  try {
    const values = await redisInterface.readValueFromKey(token);
    const jsonValues = JSON.parse(values);
    return jsonValues;
  } catch (err) {
    console.error("Token verification failed:", err);
    return null;
  }
}

function extractJwtFromCookie(cookie) {
  const startIndex = cookie.indexOf("jwt=");
  if (startIndex === -1) {
    return null;
  }
  let endIndex = cookie.indexOf(";", startIndex);
  if (endIndex === -1) {
    endIndex = cookie.length;
  }
  //extract from index to the first ;
  let jwt = cookie.slice(startIndex + 4, endIndex);
  return jwt;
}

app.get("/connecttopodinport/:jwt", async (req, res) => {
  const jwt = req.params.jwt;
  const check = await checkTokenPresence(jwt);
  if (check) {
    //create cookie with jwt to return to client
    res.cookie("jwt", jwt, { maxAge: 10800000 });
  }
  res.redirect("/");
});

app.use(async (req, res, next) => {
  try {
    if (!req.path.startsWith("/connecttopodinport/")) {
      //extract jwtstring from cookie
      const jwt = extractJwtFromCookie(req.headers.cookie);
      if (!jwt) {
        return res.status(401).send("Unauthorized");
      }
      const values = await getValuesFromToken(jwt);
      const target = `http://${values.service}.${values.namespace}.svc.cluster.local:${values.port}`;
      let proxyInstance = null;
      if (proxyCache.has(target)) {
        proxyInstance = proxyCache.get(target);
      } else {
        proxyInstance = proxy(target);
        proxyCache.set(target, proxyInstance);
      }
      console.log("URL:", target, req.path);
      proxyInstance(req, res, next, (err) => {
        console.error(err);
        res.status(500).send("Internal server error");
      });
      setTimeout(() => {
        proxyCache.delete(target);
      }, 10800000);
    } else {
      console.log("No proxy for this request");
      next();
    }
  } catch (err) {
    console.error(err);
    res.status(500).send("Internal server error");
  }
});

app.listen(proxyServerPort, () => {
  console.log(`Proxy server listening on port ${proxyServerPort}`);
});
