const yaml = require('js-yaml');

function createContentString(content) {
    let contentString = '';
    contentString += content.password + '\n';

    if(content['username']) {
        contentString += `username: ${content['username']}\n`;
    }
    if(content['url']) {
        contentString += `url: ${content['url']}\n`;
    }

    if(content['otpToken']) {
        contentString += `otpauth://totp/totp-secret?secret=${content['otpToken']}&issuer=totp-secret\n`;
    }

    let excludes = ['name', 'username', 'url', 'otpToken', 'password']

    for (const [key, value] of Object.entries(content)) {
        if(!excludes.includes(key)) {
            contentString += `${key}: ${value}\n`;
        }
    }
    return contentString;
}

const getCredValue = function (content, key) {
    if (!content.includes(key + ':')) {
        return '';
    }
    const sp = content.split(key + ':')[1];
    if (!sp) {
        return '';
    }
    return sp.split('\n')[0];
}

const getCredentialObject = function (content) {
    const lines = content.split('\n');
    const keyValuePairs = {
        password : lines.shift()
    };

    lines.forEach(line => {
        const key = line.split(":")[0];
        const value = line.split(key + ":")[1];

        if (key && value) {
            keyValuePairs[key] = value;
        }
    });

    const otpUrl = getCredValue(content, 'otpauth');
    if (otpUrl && otpUrl.length > 0) {
        const otpInfo = getOtpInfo(otpUrl);
        if (otpInfo) {
            keyValuePairs["otpToken"] = otpInfo.otpToken;
            keyValuePairs["otp"] = otpInfo.otp;
            keyValuePairs["remainingTime"] = otpInfo.remainingTime;
        }
    }
    return keyValuePairs

}
