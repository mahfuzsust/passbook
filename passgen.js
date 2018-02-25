var generator = require('generate-password');
var config = require("./data/passwordrule.json");

var generatePassword = function() {
    return password = generator.generate(config);
};

exports.generatePassword = generatePassword;