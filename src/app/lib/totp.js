const speakeasy = require('speakeasy');
const url = require('url');

const generateOTP = function (secret) {
    return speakeasy.totp({
        secret: secret,
        encoding: 'base32'
    });
}

const getRemainingTime = (secret, otp) => {
    const verifyDelta = speakeasy.totp.verifyDelta({
        secret: secret,
        encoding: 'base32',
        token: otp,
        window: 1, // You can adjust the window size based on your requirements
    });

    

    const now = Math.floor(Date.now() / 1000);
    const nextTOTPTimestamp = Math.ceil(now / 30) * 30;
    const remainingTime = nextTOTPTimestamp - now;
    return remainingTime;
}

const getOtpInfo = function (otpUrl) {
    const secret = url.parse(otpUrl, { parseQueryString: true }).query.secret;
    if (secret && secret.length > 0) {
        const otp = generateOTP(secret);
        const remainingTime = getRemainingTime(secret, otp);
        return {
            otpToken: secret,
            otp: otp,
            remainingTime: remainingTime,
        };
    }
    return null;
}