type Point3 = {
    x: number;
    y: number;
    z: number;
};
type Point2 = {
    x: number;
    y: number;
};
type Face = {
    points: Point2[];
    z: number;
    tint: number;
};
type Cluster = {
    x: number;
    y: number;
    n: number;
    depth: number;
    side: number;
    phase: number;
    alpha: number;
    parallax: number;
};
/** Approved geometric brand background. All coordinates use a 1600 × 1000 design space. */
export function createMofangBackground(ctx: CanvasRenderingContext2D) {
    let activePage: 'home' | 'login' = 'home';
    let capturing = false;
    const brandPointer = { x: 0, y: 0, targetX: 0, targetY: 0 };
    function brandRotate(v: Point3, ax: number, ay: number, az: number) { const cx = Math.cos(ax), sx = Math.sin(ax), cy = Math.cos(ay), sy = Math.sin(ay), cz = Math.cos(az), sz = Math.sin(az); const y = v.y * cx - v.z * sx, z = v.y * sx + v.z * cx, x = v.x * cy + z * sy, z2 = -v.x * sy + z * cy; return { x: x * cz - y * sz, y: x * sz + y * cz, z: z2 }; }
    function brandProject(v: Point3, cx: number, cy: number) { const p = 1050 / (1050 + v.z); return { x: cx + v.x * p, y: cy + v.y * p, z: v.z }; }
    function brandPolygon(points: Point2[]) { ctx.beginPath(); points.forEach((p, i) => i ? ctx.lineTo(p.x, p.y) : ctx.moveTo(p.x, p.y)); ctx.closePath(); }
    function brandCluster(t: number, light: boolean, settings: Cluster) { const wave = (1 - Math.cos(t * .00035 + settings.phase)) * 0.5, spacing = settings.side * (1.23 + wave * .67), ax = -.43 + Math.sin(t * .00009 + settings.phase) * .12, ay = .62 + t * .000055 + settings.phase * .18, az = -.08 + Math.sin(t * .00008 + settings.phase) * .08, faces: Face[] = []; const faceIndices = [[0, 1, 2, 3], [4, 7, 6, 5], [0, 4, 5, 1], [1, 5, 6, 2], [2, 6, 7, 3], [3, 7, 4, 0]]; for (let x = 0; x < settings.n; x++)
        for (let y = 0; y < settings.n; y++)
            for (let z = 0; z < settings.depth; z++) {
                if (settings.n === 3 && x === 1 && y === 1 && z === 1)
                    continue;
                const bx = (x - (settings.n - 1) * .5) * spacing, by = (y - (settings.n - 1) * .5) * spacing, bz = (z - (settings.depth - 1) * .5) * spacing;
                const s = settings.side * .48;
                const positions = [[-s, -s, -s], [s, -s, -s], [s, s, -s], [-s, s, -s], [-s, -s, s], [s, -s, s], [s, s, s], [-s, s, s]].map(([vx, vy, vz]) => brandProject(brandRotate({ x: bx + vx, y: by + vy, z: bz + vz }, ax, ay, az), settings.x + brandPointer.x * settings.parallax, settings.y + Math.sin(t * .00025 + settings.phase) * 9 + brandPointer.y * settings.parallax));
                faceIndices.forEach((indices, i) => { faces.push({ points: indices.map(k => positions[k]), z: indices.reduce((sum, k) => sum + positions[k].z, 0) / 4, tint: i }); });
            } faces.sort((a, b) => b.z - a.z); for (const face of faces) {
        const p = face.points, gradient = ctx.createLinearGradient(p[0].x, p[0].y, p[2].x, p[2].y);
        const opacity = settings.alpha * (face.tint % 2 === 0 ? 1 : .72);
        gradient.addColorStop(0, light ? 'rgba(164,205,237,' + (opacity * .54) + ')' : 'rgba(80,151,222,' + (opacity * .36) + ')');
        gradient.addColorStop(1, light ? 'rgba(213,234,249,' + (opacity * .14) + ')' : 'rgba(34,101,169,' + (opacity * .08) + ')');
        brandPolygon(p);
        ctx.fillStyle = gradient;
        ctx.fill();
        ctx.lineWidth = .75;
        ctx.strokeStyle = light ? 'rgba(92,144,187,' + (opacity * .84) + ')' : 'rgba(97,173,231,' + (opacity * .8) + ')';
        ctx.stroke();
    } }
    function brandWireCube(t: number, light: boolean, cx: number, cy: number, size: number, phase: number) { const s = size * .5, points = [[-s, -s, -s], [s, -s, -s], [s, s, -s], [-s, s, -s], [-s, -s, s], [s, -s, s], [s, s, s], [-s, s, s]].map(([x, y, z]) => brandProject(brandRotate({ x, y, z }, -.38 + Math.sin(t * .00007) * .08, .52 + t * .000025 + phase, -.12), cx + brandPointer.x * 7, cy + brandPointer.y * 7)); const edges = [[0, 1], [1, 2], [2, 3], [3, 0], [4, 5], [5, 6], [6, 7], [7, 4], [0, 4], [1, 5], [2, 6], [3, 7]]; ctx.lineWidth = .8; ctx.strokeStyle = light ? 'rgba(98,149,187,.20)' : 'rgba(80,146,206,.19)'; for (const [a, b] of edges) {
        ctx.beginPath();
        ctx.moveTo(points[a].x, points[a].y);
        ctx.lineTo(points[b].x, points[b].y);
        ctx.stroke();
    } const index = Math.floor(t / 2600 + phase * 2) % edges.length, [a, b] = edges[index], u = t / 2600 % 1; const x = points[a].x + (points[b].x - points[a].x) * u, y = points[a].y + (points[b].y - points[a].y) * u; ctx.beginPath(); ctx.arc(x, y, 2.1, 0, Math.PI * 2); ctx.fillStyle = light ? 'rgba(68,135,196,.56)' : 'rgba(112,199,247,.58)'; ctx.fill(); ctx.beginPath(); ctx.arc(x, y, 7, 0, Math.PI * 2); ctx.strokeStyle = light ? 'rgba(68,135,196,.12)' : 'rgba(112,199,247,.12)'; ctx.stroke(); }
    function brandFloor(t: number, light: boolean) { const project = (i: number, j: number) => ({ x: 800 + (i - j) * 59, y: 850 + (i + j) * 24 }); ctx.save(); ctx.beginPath(); ctx.rect(0, 584, 1600, 416); ctx.clip(); ctx.lineWidth = .6; ctx.strokeStyle = light ? 'rgba(103,154,195,.14)' : 'rgba(66,126,179,.13)'; for (let i = -20; i <= 20; i++) {
        for (const reverse of [false, true]) {
            const a = reverse ? project(-17, i) : project(i, -17), b = reverse ? project(17, i) : project(i, 17);
            ctx.beginPath();
            ctx.moveTo(a.x, a.y);
            ctx.lineTo(b.x, b.y);
            ctx.stroke();
        }
    } for (let k = 0; k < 6; k++) {
        const j = -9 + k * 3.4, s = ((t * .00075 + k * 4.6) % 26) - 13;
        const a = project(s, j), b = project(s + 2.8, j), gradient = ctx.createLinearGradient(a.x, a.y, b.x, b.y);
        gradient.addColorStop(0, light ? 'rgba(53,135,194,0)' : 'rgba(85,175,239,0)');
        gradient.addColorStop(.65, light ? 'rgba(53,135,194,.34)' : 'rgba(85,175,239,.38)');
        gradient.addColorStop(1, light ? 'rgba(53,135,194,0)' : 'rgba(85,175,239,0)');
        ctx.beginPath();
        ctx.moveTo(a.x, a.y);
        ctx.lineTo(b.x, b.y);
        ctx.lineWidth = 1.5;
        ctx.strokeStyle = gradient;
        ctx.stroke();
        const i = Math.floor(s), jj = Math.floor(j), corners = [project(i, jj), project(i + 1, jj), project(i + 1, jj + 1), project(i, jj + 1)];
        brandPolygon(corners);
        ctx.fillStyle = light ? 'rgba(91,164,214,.045)' : 'rgba(57,139,211,.06)';
        ctx.fill();
    } ctx.restore(); }
    function brandProtect(x: number, y: number, w: number, h: number, alpha: number, blur: number) { ctx.save(); ctx.globalCompositeOperation = 'destination-out'; ctx.fillStyle = 'rgba(0,0,0,' + alpha + ')'; ctx.shadowColor = 'rgba(0,0,0,' + alpha + ')'; ctx.shadowBlur = blur; ctx.beginPath(); ctx.roundRect(x, y, w, h, 32); ctx.fill(); ctx.restore(); }
    function drawBrandBackground(t: number, light: boolean) { if (capturing) {
        brandPointer.x = 0;
        brandPointer.y = 0;
    }
    else {
        brandPointer.x += (brandPointer.targetX - brandPointer.x) * .035;
        brandPointer.y += (brandPointer.targetY - brandPointer.y) * .035;
    } const focus = false; ctx.save(); brandFloor(t, light); brandWireCube(t, light, 1470, 794, 390, .7); brandWireCube(t, light, -70, 236, 260, 1.7); brandCluster(t, light, { x: 1510, y: 355, n: 3, depth: 3, side: 43, phase: .3, alpha: light ? .42 : .35, parallax: 10 }); brandCluster(t, light, { x: 56, y: 754, n: 2, depth: 2, side: 43, phase: 2.4, alpha: light ? .36 : .32, parallax: 7 }); brandCluster(t, light, { x: 975, y: 45, n: 2, depth: 1, side: 27, phase: 4.4, alpha: light ? .26 : .24, parallax: 5 }); if (!focus) {
        if (activePage === 'home') {
            brandProtect(98, 174, 655, 410, .91, 86);
            brandProtect(145, 665, 1325, 66, .78, 40);
        }
        else {
            brandProtect(100, 157, 635, 221, .91, 70);
            brandProtect(977, 182, 446, 620, .76, 75);
            brandProtect(140, 795, 688, 72, .72, 40);
        }
    } ctx.restore(); }
    return {
        draw(time: number, light: boolean, page: 'home' | 'login', paused: boolean) {
            activePage = page;
            capturing = paused;
            drawBrandBackground(time, light);
            for (let i = 0; i < 76; i++) {
                const x = ((Math.sin(i * 127.1 + 311.7) * 43758.5453 % 1 + 1) % 1) * 1600;
                const y = ((Math.sin(i * 83.3 + 19) * 19341.351 % 1 + 1) % 1) * 1000;
                ctx.beginPath();
                ctx.fillStyle = light ? 'rgba(49,110,172,.14)' : 'rgba(125,180,231,.21)';
                ctx.arc(x + Math.sin(time * .00015 + i) * 3, y, .55, 0, Math.PI * 2);
                ctx.fill();
            }
        },
        pointer(x: number, y: number) { brandPointer.targetX = x; brandPointer.targetY = y; },
    };
}
