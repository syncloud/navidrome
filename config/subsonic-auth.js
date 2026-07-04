function subsonicBasic(r) {
    var req = r.parent || r;
    var existing = req.headersIn['Authorization'];
    if (existing) {
        return existing;
    }
    var u = req.args.u;
    var p = req.args.p;
    if (!u || !p) {
        return '';
    }
    if (p.substr(0, 4) === 'enc:') {
        p = Buffer.from(p.substr(4), 'hex').toString('utf8');
    }
    return 'Basic ' + Buffer.from(u + ':' + p).toString('base64');
}

export default { subsonicBasic };
