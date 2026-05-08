(function () {
    // Tab switching (used on upload page)
    var tabs = document.querySelectorAll('[data-tab-target]');
    if (tabs.length > 0) {
        tabs.forEach(function (btn) {
            btn.addEventListener('click', function () {
                var target = btn.getAttribute('data-tab-target');
                document.querySelectorAll('[data-tab-target]').forEach(function (b) {
                    b.classList.toggle('active', b === btn);
                });
                document.querySelectorAll('[data-tab-panel]').forEach(function (p) {
                    p.classList.toggle('active', p.getAttribute('data-tab-panel') === target);
                });
            });
        });
    }

    // Render line numbers in paste view
    var pasteContent = document.getElementById('pasteContent');
    if (pasteContent) {
        var code = pasteContent.querySelector('code');
        if (code) {
            var text = code.textContent;
            var lines = text.split('\n');
            if (lines.length > 1 && lines[lines.length - 1] === '') {
                lines.pop();
            }
            code.innerHTML = '';
            lines.forEach(function (line) {
                var span = document.createElement('span');
                span.className = 'paste-line';
                span.textContent = line;
                code.appendChild(span);
            });
        }
    }
})();
