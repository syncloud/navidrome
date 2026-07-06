function parseQuery(qs) {
    var out = {};
    var parts = qs.split('&');
    for (var i = 0; i < parts.length; i++) {
        var eq = parts[i].indexOf('=');
        if (eq < 0) {
            continue;
        }
        out[parts[i].substring(0, eq)] = decodeURIComponent(parts[i].substring(eq + 1));
    }
    return out;
}

function subsonicBasic(r) {
    var existing = r.headersIn['Authorization'];
    if (existing) {
        return existing;
    }
    var uri = r.variables.request_uri || '';
    var q = uri.indexOf('?');
    if (q < 0) {
        return '';
    }
    var params = parseQuery(uri.substring(q + 1));
    var u = params.u;
    var p = params.p;
    if (!u || !p) {
        return '';
    }
    if (p.substr(0, 4) === 'enc:') {
        p = Buffer.from(p.substr(4), 'hex').toString('utf8');
    }
    return 'Basic ' + Buffer.from(u + ':' + p).toString('base64');
}

export default { subsonicBasic };
