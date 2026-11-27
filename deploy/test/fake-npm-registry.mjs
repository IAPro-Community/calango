// Registro npm falso, só para o teste de ponta a ponta do Maestro no VPS (deploy/test/maestro-vps-e2e.sh).
// Atende como https://registry.npmjs.org (alias de rede + CA de teste) e repassa tudo ao registro real,
// exceto o pacote do Maestro: no packument dele acrescenta as versões cujos .tgz estão em EXTRA_DIR
// (ex.: uma 1.1.2 boa e uma 1.1.3 com verify quebrado) e serve esses tarballs. EXTRA_DIR é relido a
// cada pedido, então o teste publica uma versão nova só copiando o arquivo. Nunca usar fora do teste.
import { createHash } from 'node:crypto';
import { readdirSync, readFileSync } from 'node:fs';
import https from 'node:https';
import { join } from 'node:path';

const host = 'registry.npmjs.org';
const pkg = '@iapro/orquestrador-maestro-cli';
const tarballPrefix = `/${pkg}/-/orquestrador-maestro-cli-`;
const { UPSTREAM_IP, EXTRA_DIR, TLS_KEY, TLS_CERT } = process.env;

function extras() {
  const found = new Map();
  for (const name of readdirSync(EXTRA_DIR)) {
    const match = /^orquestrador-maestro-cli-(\d+\.\d+\.\d+)\.tgz$/.exec(name);
    if (match) found.set(match[1], join(EXTRA_DIR, name));
  }
  return found;
}

function upstream(req, path, headers) {
  return https.request({ host: UPSTREAM_IP, servername: host, port: 443, method: req.method, path, headers });
}

function packument(req, res) {
  const headers = { ...req.headers, host, 'accept-encoding': 'identity' };
  delete headers['if-none-match'];
  delete headers['if-modified-since'];
  const up = upstream(req, req.url, headers);
  up.on('response', (r) => {
    const chunks = [];
    r.on('data', (c) => chunks.push(c));
    r.on('end', () => {
      const doc = JSON.parse(Buffer.concat(chunks).toString('utf8'));
      const base = doc.versions['1.1.1'];
      for (const [version, file] of extras()) {
        const data = readFileSync(file);
        doc.versions[version] = {
          ...base,
          version,
          dist: {
            tarball: `https://${host}${tarballPrefix}${version}.tgz`,
            shasum: createHash('sha1').update(data).digest('hex'),
            integrity: `sha512-${createHash('sha512').update(data).digest('base64')}`,
          },
        };
        if (doc.time) doc.time[version] = new Date().toISOString();
      }
      const body = JSON.stringify(doc);
      res.writeHead(200, { 'content-type': r.headers['content-type'] || 'application/json', 'cache-control': 'no-store' });
      res.end(body);
      console.log(`packument ${pkg}: ${Object.keys(doc.versions).filter((v) => v.startsWith('1.1.')).join(' ')}`);
    });
  });
  up.on('error', (err) => {
    res.writeHead(502);
    res.end(String(err));
  });
  up.end();
}

const server = https.createServer({ key: readFileSync(TLS_KEY), cert: readFileSync(TLS_CERT) }, (req, res) => {
  const path = decodeURIComponent(req.url.split('?')[0]);
  if (path === `/${pkg}`) {
    packument(req, res);
    return;
  }
  if (path.startsWith(tarballPrefix)) {
    const version = path.slice(tarballPrefix.length).replace(/\.tgz$/, '');
    const file = extras().get(version);
    if (file) {
      console.log(`tarball ${version} (fake)`);
      res.writeHead(200, { 'content-type': 'application/octet-stream' });
      res.end(readFileSync(file));
      return;
    }
  }
  const up = upstream(req, req.url, { ...req.headers, host });
  up.on('response', (r) => {
    res.writeHead(r.statusCode, r.headers);
    r.pipe(res);
  });
  up.on('error', (err) => {
    res.writeHead(502);
    res.end(String(err));
  });
  req.pipe(up);
});
server.listen(443, () => console.log(`fake registry for ${host} -> ${UPSTREAM_IP}`));
