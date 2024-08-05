//da implementare controllo cf in server ldap
function checkCfLdap(cf,pw){
  return new Promise((resolve,reject) => {
    const ldap = require('ldapjs');
    const ldapUrl = 'ldap://AD-UNICT-DC1.unict.ad'; // AD server url
    const ldapUrl2='ldap://AD-UNICT-DC2.unict.ad'
    const baseDn = 'DC=unict,DC=ad'; 
    const userDn = `CN=${cf},OU=Studenti,${baseDn}`;
    const ldap_client = ldap.createClient({
      url: [ldapUrl,ldapUrl2]
    });
    ldap_client.on('error', (err) => {
      console.error('Errore durante la connessione:', err.message);
      return reject(err);
    });
    // Esegui il bind (autenticazione) dell'utente
    ldap_client.bind(userDn, pw, (err) => {
      if (err) {
        resolve(false);
      } else {
        resolve(true);
      }
      // Chiudi la connessione dopo il bind
      ldap_client.unbind((unbindErr) => {
        if (unbindErr) {
          console.error('Errore durante la disconnessione:', unbindErr.message);
        }
      });
    });
  });
}
module.exports = {checkCfLdap};