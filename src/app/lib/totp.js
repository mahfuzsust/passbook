const OTPAuth = require('otpauth');

const generateOTP = function (otpUrl) {
    const totp = OTPAuth.URI.parse(otpUrl);
    return totp.generate();
}

const getRemainingTime = () => {
    const now = Math.floor(Date.now() / 1000);
    const nextTOTPTimestamp = Math.ceil(now / 30) * 30;
    const remainingTime = nextTOTPTimestamp - now;
    return remainingTime;
}

const getOtpInfo = function (otpUrl) {
    if (otpUrl && otpUrl.length > 0) {
        const otp = generateOTP(otpUrl);
        const remainingTime = getRemainingTime();
        return {
            otpUrl: otpUrl,
            otp: otp,
            remainingTime: remainingTime,
        };
    }
    return null;
}