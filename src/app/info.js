const version = process.argv.filter((arg) => arg.startsWith('--version='))[0]?.split('=')?.[1]
document.getElementById('version').innerHTML = "Version: " + version;