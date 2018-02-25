var bcrypt = require("bcryptjs");
var Datastore = require('nedb')
  , users = new Datastore({ filename: './data/users.db', autoload: true })
  , books = new Datastore({ filename: './data/books.db', autoload: true })
  , credentials = new Datastore({ filename: './data/credentials.db', autoload: true });

// book
var getAllBook = function (userId, callback) {
	books.find({userId: userId}, callback);
};
var addBook = function (book, callback) {
  book["created"] = new Date();
  book["updated"] = null;
	books.insert(book, callback);
};
var editBook = function (book, callback) {
	books.update({ _id: book._id }, { 
    $set: { name: book.name, updated: new Date()  } 
  }, {returnUpdatedDocs: true}, callback);
};
var deleteBook = function (bookId, callback) {
	books.remove({_id: bookId}, function(err, num){
    credentials.remove({bookId: bookId}, { multi: true }, callback);
  });
};

//user 
var getUser = function(username, callback) {
  users.findOne({ username: username }, callback);
};
var addUser = function(user, callback) {
  users.insert({username:user.username, password: bcrypt.hashSync(user.password, 8)}, callback);
};
var updatePassword = function(user, callback) {
  users.update({ username: user.username }, { 
    $set: { password: bcrypt.hashSync(user.password, 8), updated: new Date()  } 
  }, {returnUpdatedDocs: true}, callback);
};

// credential
var addCredential = function (credential, callback) {
  credential["created"] = new Date();
  credential["updated"] = null;
	credentials.insert(credential, callback);
};
var editCredential = function (credential, callback) {
	credentials.update({ _id: credential._id }, { 
    $set: { name: credential.name, password: credential.password, url: credential.url, updated: new Date()  } 
  }, callback);
};
var deleteCredential = function (credentialId, callback) {
	credentials.remove({_id: credentialId}, callback);
};
var getAllCredential = function (bookId, callback) {
	credentials.find({bookId: bookId}, callback);
};

exports.getAllBook = getAllBook;
exports.addBook = addBook;
exports.getUser = getUser;
exports.getAllCredential = getAllCredential;
exports.addCredential = addCredential;
exports.editCredential = editCredential;
exports.deleteCredential = deleteCredential;
exports.editBook = editBook;
exports.deleteBook = deleteBook;
exports.addUser = addUser;