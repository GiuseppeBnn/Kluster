const axios = require('axios');
const multer = require('multer');
const FormData = require('form-data'); 

const upload = multer();

const goServerIp = "localhost";
const goServerPort = 9000;
const protocol = "http";

//TODO: insert try and catch where necessary to handle errors

function receiveAndCheckFiles(req, res, next) {
    upload.fields([{ name: 'yaml' }, { name: 'file' }])(req, res, function (err) {
        if (err) {
            return res.status(400).send('Errore durante il caricamento dei file.');
        }
        const yamlFile = req.files.yaml[0];
        const zipFile = req.files.file[0];

        if (yamlFile.size > 1 * 1024 * 1024) { // 1MB
            return res.status(400).send('Il file YAML supera la dimensione massima consentita di 1MB.');
        }
        if (zipFile.size > 10 * 1024 * 1024) { // 10MB
            return res.status(400).send('Il file ZIP supera la dimensione massima consentita di 10MB.');
        }
        next();
    });
}

async function forwardToGoServer(req, res) {
    
    const form = new FormData(); 
    form.append('name', req.body.name);
    form.append('yamlFile', req.files.yaml[0].buffer, "values.yaml");
    form.append('zipFile', req.files.file[0].buffer, "mount.zip");
    try {
        const formHeaders = form.getHeaders();
        const response = await axios.post(`${protocol}://${goServerIp}:${goServerPort}/upload`, form, {
            headers: {
                ...formHeaders,
                Authorization: `${req.session.token}`
            }
        });
        console.log(response.data); // da cambiare con gestione risposta
    } catch (error) {
        console.error(error);
        res.status(500).send('Errore durante l\'inoltro dei dati al server Go.');
    }
}

async function getListOfCharts(token){
    const response = await axios.get(`${protocol}://${goServerIp}:${goServerPort}/list`, {
        headers: {
            Authorization: token
        }
    });
    return response.data;
}

async function startChart(chartJwt, token ){
    try{
        const response = await axios.get(`${protocol}://${goServerIp}:${goServerPort}/install`, {
            headers: {
                Authorization: token,
                referredChart: chartJwt
            }
        });
        return response.data;
    } catch (error) {
        console.error(error);
        return error;
    }
}
async function deleteChart(chartJwt, token ){
    try{
        const response = await axios.get(`${protocol}://${goServerIp}:${goServerPort}/delete`, {
            headers: {
                Authorization: token,
                referredChart: chartJwt
            }
        });
        return response.data;
    } catch (error) {
        console.error(error);
        return error;
    }
}
function stopChart(chartName, token){
    axios.get(`${protocol}://${goServerIp}:${goServerPort}/stop`, {
        headers: {
            Authorization: token,
            referredChart: chartName
        }
    });
}


module.exports = { receiveAndCheckFiles, forwardToGoServer, getListOfCharts, startChart , deleteChart, stopChart};