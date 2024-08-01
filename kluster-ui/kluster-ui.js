const express = require("express");
const session = require("express-session");
const jwt = require("jsonwebtoken");
const ldapInterface = require("./ldapInterface");
const RedisInterface = require("./redisInterface");
const helmInterface = require("./helmInterface");
const redisInterface = new RedisInterface("192.168.1.40", 30207); //da cambiare con ip e porta corretti
const app = express();

app.set("view engine", "ejs");
app.use(express.urlencoded({ extended: true }));
app.use(express.json());
app.use(
  session({
    secret: "segreto_da_cambiare", //secret for session, change it in production
    resave: false,
    saveUninitialized: false,
    cookie: { maxAge: 3600000 },
  })
);
app.use(express.static(__dirname + "/build"));

const port = 3000;
const jwtSecret = "segreto_da_cambiare_jwt"; //secret for jwt token, change it

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
  // da implementare controllo cf in server ldap con password e tutto
  if (!ldapInterface.checkCF(req.body.cf)) {
    return res.status(401).json({ error: "CF not valid" });
  }
  const token = createTokenFromCf(req.body.cf);
  try {
    await redisInterface.setKeyValue(token, req.body.cf, expiry=3600);
    req.session.token = token;
    res.redirect("/dashboard");
  } catch (err) {
    console.error("Error setting token in Redis:", err);
    res.status(500).json({ error: "Internal server error" });   //tutti gli errori devono essere gestiti con un unico oggetto json
  }
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
  res.render("login");
});

app.get("/dashboard", checkToken, (req, res) => {
  res.render("dashboard");
} );
  
app.get("/charts-status", checkToken,async (req, res) => {
  let charts = await helmInterface.getListOfCharts(req.session.token);
  res.render("charts-status", { charts: charts });
});
app.post("/upload", checkToken, helmInterface.receiveAndCheckFiles, helmInterface.forwardToGoServer);


app.get("/upload", checkToken, (req, res) => {
  res.render("upload");
});

app.get("/chart-status/:chartJwt", checkToken, async (req, res) => {
  const charts = await helmInterface.getListOfCharts(req.session.token);
  const chart = charts.find((chart) => chart.jwt == req.params.chartJwt);
  console.log("CHART", chart);
  res.render("components/layout/dp-dash-el", { chart: chart });
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

