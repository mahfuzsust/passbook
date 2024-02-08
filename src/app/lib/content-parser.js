const yaml = require('js-yaml');

function createContentString(content) {
    let contentString = '';
    
    if (content['password']) {
        contentString += content['password'] + '\n';
    } else {
        contentString += '\n';
    }
    
    if (content['username']) {
        contentString += `username: ${content['username']}\n`;
    }
    if (content['url']) {
        contentString += `url: ${content['url']}\n`;
    }

    if (content['otpToken']) {
        contentString += `otpauth://totp/totp-secret?secret=${content['otpToken']}&issuer=totp-secret\n`;
    }

    let excludes = ['name', 'username', 'url', 'otpToken', 'password']

    for (const [key, value] of Object.entries(content)) {
        if (!excludes.includes(key) && value) {
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

const getNote = function(currentValue, value) {
    if (!currentValue) {
        currentValue = '';
    }
    if (!value) {
        return currentValue;
    }
    if (currentValue.length > 0) {
        currentValue += '\n';
    }
    currentValue += value;   
    return currentValue; 
}

const getCredentialObject = function (content) {
    const lines = content.split('\n');
    const keyValuePairs = {
        password: lines.shift()
    };

    const fields = ['username', 'url', 'notes', 'otpauth'];

    let otpUrl = "";
    lines.forEach(line => {
        const isKeyValuePair = line.includes(':');
        const key = line.split(":")[0];
        let value = line.split(key + ":")[1];
        if(value) {
            value = value.trim();
        }

        if (isKeyValuePair) {
            if (fields.includes(key)) {
                if(key === 'notes') {
                    keyValuePairs[key] = getNote(keyValuePairs[key], value);
                } else if(key === 'otpauth') {
                    otpUrl = line;
                } else {
                    keyValuePairs[key] = value;
                }
            } else {
                keyValuePairs['notes'] = getNote(keyValuePairs['notes'], line);
            }
        } else {
            keyValuePairs['notes'] = getNote(keyValuePairs['notes'], line);
        }
    });

    if (otpUrl && otpUrl.length > 0) {
        const otpInfo = getOtpInfo(otpUrl);
        if (otpInfo) {
            keyValuePairs["otpUrl"] = otpInfo.otpUrl;
            keyValuePairs["otp"] = otpInfo.otp;
            keyValuePairs["remainingTime"] = otpInfo.remainingTime;
        }
    }
    keyValuePairs["raw"] = content;
    return keyValuePairs

}
