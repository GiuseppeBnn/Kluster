//da implementare controllo cf in server ldap
function checkCF(cf) {
    if (!cf || cf === "") {
        return false;
    }
    return true;
}

module.exports = {checkCF };