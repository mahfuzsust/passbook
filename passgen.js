var generator = require('generate-password');
var config = require("./passwordrule.json");

var generatePassword = function() {
    return password = generator.generate(config);
};

exports.generatePassword = generatePassword;