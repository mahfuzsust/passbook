var bcrypt = require("bcryptjs");
var Datastore = require('nedb')
  , users = new Datastore({ filename: './data/users.db', autoload: true })
  , books = new Datastore({ filename: './data/books.db', autoload: true })
  , credentials = new Datastore({ filename: './data/credentials.db', autoload: true });

//users.insert({username:"mahfuz", password: bcrypt.hashSync('pass', 8)});

var getAllBook = function (callback) {
	books.find({}, callback);
};
var addBook = function (book, callback) {
	books.insert({name: book, created: new Date()}, callback);
};
var addBook = function (book, callback) {
	books.insert({name: book, created: new Date()}, callback);
};
var getUser = function(username, callback) {
  users.findOne({ username: username }, callback);
};
var addCredential = function (credential, callback) {
  credential["created"] = new Date();
	credentials.insert(credential, callback);
};
var getAllCredential = function (bookId, callback) {
	credentials.find({bookId: bookId}, callback);
};

exports.getAllBook = getAllBook;
exports.addBook = addBook;
exports.getUser = getUser;
exports.getAllCredential = getAllCredential;
exports.addCredential = addCredential;