const showToast = function (mdtoast, message, timeout = 2000) {
    mdtoast.show(
        mdtoast.simple()
            .textContent(message)
            .position('top right')
            .hideDelay(timeout))
        .then(function () {
        }).catch(function () {
        });
}