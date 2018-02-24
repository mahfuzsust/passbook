var Datastore = require('nedb')
  , users = new Datastore({ filename: './data/users.db', autoload: true })
  , books = new Datastore({ filename: './data/books.db', autoload: true })
  , credentials = new Datastore({ filename: './data/credentials.db', autoload: true });


var getAllBook = function () {
	books.find({}, function (err, docs) {
		console.log(docs);
	 	return docs;
	});
};
var addBook = function (book) {
	books.insert({name: book}, function (err, newDoc) {
	});
};

exports.getAllBook = getAllBook;
exports.addBook = addBook;