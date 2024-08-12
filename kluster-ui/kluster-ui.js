const express = require("express");
const session = require("express-session");
const jwt = require("jsonwebtoken");
const ldapInterface = require("./ldapInterface");
const RedisInterface = require("./redisInterface");
const helmInterface = require("./helmInterface");
require("dotenv").config();
const redisInterface = new RedisInterface(
  process.env.REDIS_H,
  process.env.REDIS_P
);
const app = express();
const jwtSecret = process.env.JWT_SECRET;
const sessionSecret = process.env.SESSION_SECRET;
const proxyUrl = process.env.PROXY_URL;
const proxySecret = process.env.PROXY_SECRET;

app.set("view engine", "ejs");
app.use(express.urlencoded({ extended: true }));
app.use(express.json());
app.use(
  session({
    secret: sessionSecret, //secret for session, change it in production
    resave: true,
    saveUninitialized: false,
  })
);
app.use(express.static(__dirname + "/build"));

const port = 3000;

function createTokenFromCf(cf) {
  return jwt.sign({ cf }, jwtSecret, { expiresIn: "1h" });
}
async function verifyToken(token) {
  try {
    const decoded = jwt.verify(token, jwtSecret);
    const tokenExists = await redisInterface.checkTokenPresence(token);
    if (tokenExists && decoded.cf) {
      return true;
    } else {
      console.log("Token not found in Redis");
      return false;
    }
  } catch (err) {
    console.error("Token verification failed:", err);
    return null;
  }
}

async function checkToken(req, res, next) {
  const token = req.session.token;
  if (!token) {
    return res.redirect("/login");
  }
  try {
    if (await verifyToken(token)) {
      return next();
    } else {
      console.log("Token not found in Redis");
      res.redirect("/login");
    }
  } catch (err) {
    console.error("Token verification failed");
    res.redirect("/login");
  }
}

async function createAndCacheProxyJwt(chartJwt, service, port, namespace) {
  const newJwt = jwt.sign(
    { chartJwt: chartJwt, service: service, port: port, namespace: namespace },
    proxySecret,
    { expiresIn: "3h" }
  );
  const value = JSON.stringify({
    service: service,
    port: port,
    namespace: namespace,
  });
  await redisInterface.setKeyValue(newJwt, value, (expiry = 10800));
  return newJwt;
}

app.get("/", checkToken, (req, res) => {
  res.redirect("/dashboard");
});

app.post("/login", async (req, res) => {
  ldapInterface
    .checkCfLdap(req.body.cf, req.body.pw)
    .then(async (isAuth) => {
      if (true) {
        ///IMPORTANTE!!! da cambiare con if(isAuth)
        const token = createTokenFromCf(req.body.cf);
        await redisInterface.setKeyValue(token, req.body.cf, (expiry = 3600));
        req.session.token = token;
        return res.header("Authorization", token).status(201).send("ok");
      }
      res.status(400).send({ error: "Invalid credentials" });
    })
    .catch((error) => {
      console.error("Auth error:", error);
      res.render("login", { error: "Something went wrong", login: true });
    });
});

app.get("/logout", checkToken, async (req, res) => {
  const token = req.session.token;
  try {
    redisInterface.delKey(token);
  } catch (err) {
    console.error("Error deleting token from Redis:", err);
  }
  req.session.token = null;
  res.redirect("/login");
});

app.get("/login", async (req, res) => {
  if (req.session.token) {
    if (await verifyToken(req.session.token)) {
      return res.redirect("/dashboard");
    } else {
      console.log("Token not found");
      req.session.token = null;
    }
  }
  req.session.token = null;
  res.status(201).render("login", { login: true });
});

app.get("/dashboard", checkToken, (req, res) => {
  res.render("dashboard");
});

app.get("/charts-status", checkToken, async (req, res) => {
  const response = await helmInterface.getListOfCharts(req.session.token);
  const charts = JSON.parse(response.message);
  res.render("charts-status", { charts: charts, type: response.type });
});

app.post(
  "/upload",
  checkToken,
  helmInterface.receiveAndCheckFiles,
  helmInterface.forwardToGoServer
);

app.get("/upload", checkToken, (req, res) => {
  res.render("upload");
});

app.get("/chart-status/:chartJwt", checkToken, async (req, res) => {
  const response = await helmInterface.getListOfCharts(req.session.token);
  const charts = JSON.parse(response.message);
  try {
    const chart = charts.find((chart) => chart.jwt == req.params.chartJwt);
    res.render("components/layout/dp-dash-el", {
      chart: chart,
      type: response.type,
    });
  } catch (err) {
    return res.status(404).send("Chart not found");
  }
});

app.patch("/play/:chartJwt", checkToken, async (req, res) => {
  const result = await helmInterface.startChart(
    req.params.chartJwt,
    req.session.token
  );
  if (result) {
    res.send("ok");
  } else {
    res.status(400).send("error");
  }
});

app.patch("/stop/:chartName", checkToken, async (req, res) => {
  const result = await helmInterface.stopChart(
    req.params.chartName,
    req.session.token
  );
  if (result) {
    res.send("ok");
  } else {
    res.status(400).send("error");
  }
});

app.delete("/delete/:chartName", checkToken, async (req, res) => {
  const result = await helmInterface.deleteChart(
    req.params.chartName,
    req.session.token
  );
  if (result) {
    res.send("ok");
  } else {
    res.status(400).send("error");
  }
});

app.get("/details/:chartJwt", checkToken, async (req, res) => {
  res.render("details", { jwt: req.params.chartJwt });
});

app.get("/dp-details/:chartJwt", checkToken, async (req, res) => {
  const response = await helmInterface.getDetails(
    req.params.chartJwt,
    req.session.token
  );
  try {
    chart = JSON.parse(response.message);
    res.render("components/layout/dp-details-el", { chart: chart });
  } catch (err) {
    return res.status(404).send("Chart not found");
  }
});

app.get("/logs/:chartJwt/:podName", checkToken, async (req, res) => {
  res.render("logs", { jwt: req.params.chartJwt, pod: req.params.podName });
});

app.get("/dp-logs/:chartJwt/:podName", checkToken, async (req, res) => {
  const response = await helmInterface.getDeploymentLogs(
    req.params.chartJwt,
    req.params.podName,
    req.session.token
  );
  try {
    const logs = JSON.parse(response.message);
    res.render("components/layout/dp-logs-el", { chart: logs });
  } catch (err) {
    return res.status(404).send("Logs not found");
  }
});

app.get(
  "/forward-to-port/:chartJwt/:service/:port/:namespace",
  checkToken,
  async (req, res) => {
    try {
      const newJwt = await createAndCacheProxyJwt(
        req.params.chartJwt,
        req.params.service,
        req.params.port,
        req.params.namespace
      );
      const newUrl = `${proxyUrl}/connecttopodinport/${newJwt}`;
      res.redirect(newUrl); // ovviamente in futuro usare HTTPS
    } catch (err) {
      console.error("Error forwarding to port:", err);
      return res.status(500).send("Error forwarding to port");
    }
  }
);

app.listen(port, () => {
  console.log(`Listening at http://localhost:${port}`);
});
