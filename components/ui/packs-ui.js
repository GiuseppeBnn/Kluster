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
const jwtDeliverSecret = process.env.JWT_DELIVER_SECRET;

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

function createDeliverToken() {
  return jwt.sign(
    {
      role: "deliver",
      time: new Date().getTime().toString(),
    },
    jwtDeliverSecret,
    { expiresIn: "3h" }
  );
}

function createTokenFromCf(cf) {
  return jwt.sign(
    { role: "user", cf: cf, time: new Date().getTime().toString() },
    jwtSecret,
    {
      expiresIn: "1h",
    }
  );
}
function createAdminToken() {
  return jwt.sign(
    {
      role: "admin",
      time: new Date().getTime().toString(),
    },
    jwtSecret,
    { expiresIn: "3h" }
  );
}

async function verifyAdminToken(token) {
  try {
    const decoded = jwt.verify(token, jwtSecret);
    const tokenExists = await redisInterface.checkTokenPresence(token);
    if (decoded.role == "admin" && tokenExists) {
      return true;
    }
    return false;
  } catch (err) {
    console.error("Token verification failed:", err);
    return null;
  }
}

async function verifyToken(token) {
  let decoded;
  try {
    decoded = jwt.verify(token, jwtSecret);
  } catch (err) {
    return false;
  }
  const tokenExists = await redisInterface.checkTokenPresence(token);
  if (tokenExists && decoded.role) {
    return true;
  }
  return false;
}

async function checkAdminToken(req, res, next) {
  const token = req.session.token;
  if (!token) {
    return res.redirect("/login");
  }
  try {
    if (await verifyAdminToken(token)) {
      return next();
    } else {
      res.redirect("/login");
    }
  } catch (err) {
    res.redirect("/login");
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
  //CAMBIARE GESTIONE LOGIN PER CONTROLLO ADMIN
  if (req.body.cf == "admin" && req.body.pw == "admin") {
    const token = createAdminToken();
    await redisInterface.setKeyValue(token, req.body.cf, (expiry = 10800)); //da cambiare assolutamente
    req.session.token = token;
    return res.header("Authorization", token).status(201).send("ok");
  }
  ldapInterface
    .checkCfLdap(req.body.cf, req.body.pw)
    .then(async (isAuth) => {
      if (isAuth) {
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

app.get("/dashboard", checkToken, async (req, res) => {
  const isAdm = await verifyAdminToken(req.session.token);
  if (isAdm) {
    return res.redirect("/admin/dashboard");
  } else {
    res.render("dashboard");
  }
});

app.get("/charts-status", checkToken, async (req, res) => {
  const response = await helmInterface.getListOfCharts(req.session.token);
  const charts = JSON.parse(response.message);
  res
    .status(202)
    .render("charts-status", { charts: charts, type: response.type });
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
    res.status(202).render("components/layout/dp-dash-el", {
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

app.get("/md-deliver/:chart/", checkToken, async (req, res) => {
  res.render("components/layout/md-deliver", { chart: req.params.chart });
});

app.post("/deliver/", checkToken, async (req, res) => {
  try {
    let b = await redisInterface.useDeliverToken(req.body.deliveryToken);
    if (b) {
      await helmInterface.setDeliveredChart(
        req.body.chartJwt,
        req.session.token
      );
      console.log("Chart delivered", req.body.chartJwt);
      return res.status(202).send("ok");
    } else {
      return res.status(400).send("error");
    }
  } catch (err) {
    res.status(400).send("error");
  }
});

app.get("/admin/dashboard", checkAdminToken, async (req, res) => {
  res.render("admin_dashboard");
});
app.get("/admin/generate-deliver-token", checkAdminToken, async (req, res) => {
  const deliverToken = createDeliverToken();
  await redisInterface.setDeliverToken(deliverToken, (expiry = 108000));
  res.status(202).send(deliverToken);
});
app.delete("/undeliver/:chartJwt", checkAdminToken, async (req, res) => {
  try {
    await helmInterface.setUndeliveredChart(
      req.params.chartJwt,
      req.session.token
    );
    res.status(202).send("ok");
  } catch (err) {
    res.status(400).send("error");
  }
});
app.listen(port, () => {
  console.log(`Listening at http://localhost:${port}`);
});
