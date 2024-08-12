require("dotenv").config();

function checkCfLdap(cf, pw) {
  return new Promise((resolve, reject) => {
    const ldap = require("ldapjs");
    const ldapUrl = process.env.LDAP_URL1;
    const ldapUrl2 = process.env.LDAP_URL2;
    const baseDn = "DC=unict,DC=ad";
    const userDn = `CN=${cf},OU=Studenti,${baseDn}`;
    const ldap_client = ldap.createClient({
      url: [ldapUrl, ldapUrl2],
    });
    ldap_client.on("error", (err) => {
      console.error("Errore durante la connessione:", err.message);
      return reject(err);
    });
    // Esegue il bind (autenticazione) dell'utente
    ldap_client.bind(userDn, pw, (err) => {
      if (err) {
        resolve(false);
      } else {
        resolve(true);
      }
      // Chiude la connessione dopo il bind
      ldap_client.unbind((unbindErr) => {
        if (unbindErr) {
          console.error("Errore durante la disconnessione:", unbindErr.message);
        }
      });
    });
  });
}
module.exports = { checkCfLdap };
