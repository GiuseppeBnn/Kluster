const express = require("express");
const session = require("express-session");
const jwt = require("jsonwebtoken");
const ldapInterface = require("./ldapInterface");
const RedisInterface = require("./redisInterface");
const helmInterface = require("./helmInterface");
const redisInterface = new RedisInterface("192.168.1.40", 30207); //da cambiare con ip e porta corretti
const app = express();
const jwtSecret="a very big secret"
const sessionSecret="secret to change, it's important"

app.set("view engine", "ejs");
app.use(express.urlencoded({ extended: true }));
app.use(express.json());
app.use(
  session({
    secret: sessionSecret, //secret for session, change it in production
    resave: true,
    saveUninitialized: false,
    //cookie: { maxAge: 3600000 },
  })
);
app.use(express.static(__dirname + "/build"));

const port = 3000;

function createTokenFromCf(cf) {
  return jwt.sign({ cf }, jwtSecret, { expiresIn: '1h' });
}
async function verifyToken(token) {
  try {
    const decoded = jwt.verify(token, jwtSecret);
    const tokenExists = await redisInterface.checkTokenPresence(token);
    if (tokenExists && decoded.cf) {
      return true;
    }
    else {
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
    }
    else {
      console.log("Token not found in Redis");
      res.redirect("/login");
    }
    
  } catch (err) {
    console.error("Token verification failed:", err);
    res.redirect("/login");
  }
}

app.get("/", checkToken, (req, res) => {
  res.redirect("/dashboard");
});

app.post("/login", async (req, res) => {
  ldapInterface.checkCfLdap(req.body.cf,req.body.pw).then(async (isAuth) => {
  
  if (isAuth) {   ///IMPORTANTE!!! da cambiare con if(isAuth)
    const token = createTokenFromCf(req.body.cf);
    await redisInterface.setKeyValue(token, req.body.cf, expiry=3600);
    req.session.token = token;
    return res.header("Authorization", token).status(201).send("ok");
  }
  res.status(400).send({ error: "Invalid credentials" });
}).catch((error) => {
  console.error("LDAP error:", error);
  res.render("login", { error: "Something went wrong" ,login: true});
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
    }
    else {
      console.log("Token not found");
      req.session.token = null;
    }
  }
  req.session.token = null;
  res.status(201).render("login",{login: true});
});

app.get("/dashboard", checkToken, (req, res) => {res.render("dashboard");});
  
app.get("/charts-status", checkToken, async (req, res) => {
  const response= await helmInterface.getListOfCharts(req.session.token);
  const charts = JSON.parse(response.message);
  res.render("charts-status", { charts: charts , type: response.type});
});

app.post("/upload", checkToken, helmInterface.receiveAndCheckFiles, helmInterface.forwardToGoServer);


app.get("/upload", checkToken, (req, res) => {res.render("upload");});

app.get("/chart-status/:chartJwt", checkToken, async (req, res) => {
  const response = await helmInterface.getListOfCharts(req.session.token);
  const charts = JSON.parse(response.message);
  const chart = charts.find((chart) => chart.jwt == req.params.chartJwt);
  res.render("components/layout/dp-dash-el", { chart: chart, type: response.type });
});


app.patch("/play/:chartJwt",checkToken, async (req, res) => {
  await helmInterface.startChart(req.params.chartJwt, req.session.token);
  res.send("ok");
});

app.patch("/stop/:chartName", checkToken,(req, res) => {
  helmInterface.stopChart(req.params.chartName,req.session.token);
  res.send("ok");
});

app.delete("/delete/:chartName", checkToken,(req, res) => {
  helmInterface.deleteChart(req.params.chartName,req.session.token);
  res.send("ok");
});




app.listen(port, () => {
  console.log(`Listening at http://localhost:${port}`);
});

