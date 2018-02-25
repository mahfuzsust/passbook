var crypto = require('crypto'),
    algorithm = 'aes-256-ctr';

var encrypt = function(text, hash) {
    var cipher = crypto.createCipher(algorithm,hash)
    var crypted = cipher.update(text,'utf8','hex')
    crypted += cipher.final('hex');
    return crypted;
};
 
var decrypt = function(text, hash) {
    var decipher = crypto.createDecipher(algorithm,hash)
    var dec = decipher.update(text,'hex','utf8')
    dec += decipher.final('utf8');
    return dec;
};

exports.encrypt = encrypt;
exports.decrypt = decrypt;