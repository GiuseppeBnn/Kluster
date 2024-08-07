const axios = require("axios");
const multer = require("multer");
const FormData = require("form-data");
const upload = multer();

const goServerIp = "localhost";
const goServerPort = 9000;
const protocol = "http";

//TODO: insert try and catch where necessary to handle errors

function receiveAndCheckFiles(req, res, next) {
  upload.fields([{ name: "yaml" }, { name: "file" }])(req, res, function (err) {
    try {
      if (err) {
        return res.status(400).send("Errore durante il caricamento dei file.");
      }
      const yamlFile = req.files.yaml[0];
      if (typeof req.files.file !== "undefined") {
        const zipFile = req.files.file[0];
        if (zipFile.size > 10 * 1024 * 1024) {
          // 10MB
          return res
            .status(400)
            .send(
              "Il file ZIP supera la dimensione massima consentita di 10MB."
            );
        }
      }
      if (yamlFile.size > 1 * 1024 * 1024) {
        // 1MB
        return res
          .status(400)
          .send("Il file YAML supera la dimensione massima consentita di 1MB.");
      }
      next();
    } catch (error) {
      console.log(error);
      return res
        .status(400)
        .send({ type: "error", message: "File upload error" });
    }
  });
}

async function forwardToGoServer(req, res) {
  try {
    const form = new FormData();
    form.append("name", req.body.name);
    form.append("yamlFile", req.files.yaml[0].buffer, "values.yaml");
    if (typeof req.files.file !== "undefined") {
      form.append("zipFile", req.files.file[0].buffer, "mount.zip");
    }
    const formHeaders = form.getHeaders();
    const response = await axios.post(
      `${protocol}://${goServerIp}:${goServerPort}/upload`,
      form,
      {
        headers: {
          ...formHeaders,
          Authorization: `${req.session.token}`,
        },
      }
    );
  } catch (error) {
    console.log(error);
    if (
      typeof error.response == "undefined" ||
      typeof error.response.data == "undefined"
    ) {
      return res
        .status(400)
        .send({ type: "error", message: "File upload error" });
    }
    return res.status(400).send(error.response.data);
  }
  res.status(200).send("uploaded");
}

async function getListOfCharts(token) {
  try {
    const response = await axios.get(
      `${protocol}://${goServerIp}:${goServerPort}/list`,
      {
        headers: {
          Authorization: token,
        },
      }
    );
    return response.data;
  } catch (error) {
    console.log(error);
    return {
      type: "error",
      message: '{"error": "Error getting list of charts"}',
    };
  }
}

async function startChart(chartJwt, token) {
  try {
    const response = await axios.get(
      `${protocol}://${goServerIp}:${goServerPort}/install`,
      {
        headers: {
          Authorization: token,
          referredChart: chartJwt,
        },
      }
    );
    return true;
  } catch (error) {
    console.log(error);
    return false;
  }
}
async function deleteChart(chartJwt, token) {
  try {
    const response = await axios.get(
      `${protocol}://${goServerIp}:${goServerPort}/delete`,
      {
        headers: {
          Authorization: token,
          referredChart: chartJwt,
        },
      }
    );
    return true;
  } catch (error) {
    console.log(error);
    return false;
  }
}
async function stopChart(chartName, token) {
  try {
    const response = await axios.get(
      `${protocol}://${goServerIp}:${goServerPort}/stop`,
      {
        headers: {
          Authorization: token,
          referredChart: chartName,
        },
      }
    );
    return true;
  } catch (error) {
    console.log(error);
    return false;
  }
}

async function getDetails(chartJwt, token) {
  try {
    const response = await axios.get(
      `${protocol}://${goServerIp}:${goServerPort}/details`,
      {
        headers: {
          Authorization: token,
          referredChart: chartJwt,
        },
      }
    );
    return response.data;
  } catch (error) {
    console.log(error);
    return {
      type: "error",
      message: '{"error": "Error getting details of chart"}',
    };
  }
}

async function getDeploymentLogs(chartJwt, podName, token) {
  try {
    const response = await axios.get(
      `${protocol}://${goServerIp}:${goServerPort}/logs`,
      {
        headers: {
          Authorization: token,
          referredChart: chartJwt,
          podName: podName,
        },
      }
    );
    return response.data;
  } catch (error) {
    console.log(error);
    return {
      type: "error",
      message: '{"error": "Error getting logs of chart"}',
    };
  }
}

module.exports = {
  receiveAndCheckFiles,
  forwardToGoServer,
  getListOfCharts,
  startChart,
  deleteChart,
  stopChart,
  getDetails,
  getDeploymentLogs,
};
