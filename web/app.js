document.addEventListener('DOMContentLoaded', () => {
    let currentPage = 1;
    let isLoading = false;
    let hasMore = true;
    let currentMode = 'timeline'; // 'timeline' or 'search'
    let currentSearchQuery = '';

    const timelineEl = document.getElementById('timeline');
    const sentinelEl = document.getElementById('loading-sentinel');
    const searchInput = document.getElementById('search-input');

    // Theme logic
    const themeBtn = document.getElementById('theme-btn');
    const themeIcon = document.getElementById('theme-icon');
    const THEMES = ['system', 'light', 'dark'];
    const systemDarkQuery = window.matchMedia('(prefers-color-scheme: dark)');

    function updateThemeIcon(theme) {
        if (theme === 'system') {
            themeIcon.innerHTML = `<path d="M21 9C19.3529 9 17.8917 9.79647 16.9808 11.0253M21 6C18.5797 6 16.4104 7.07479 14.9434 8.77313M21 3C17.1326 3 13.7313 4.99586 11.77 8.01376M11.77 8.01376C9.72698 8.16181 8.00348 9.48869 7.25 11.25C4.7 11.6562 3 13.7572 3 16.0315C3 18.7755 5.28335 21 8.1 21L15.75 21C18.0972 21 20 19.1279 20 16.8185C20 15.1039 18.951 13.5202 17.45 12.875C17.4116 12.2181 17.2475 11.5941 16.9808 11.0253M11.77 8.01376C11.8958 8.00465 12.0229 8 12.151 8C13.1755 8 14.1323 8.28298 14.9434 8.77313M14.9434 8.77313C15.8305 9.30914 16.5435 10.0929 16.9808 11.0253" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>`;
        } else if (theme === 'light') {
            themeIcon.innerHTML = '<circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>';
        } else {
            themeIcon.innerHTML = '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>';
        }
    }

    function applyTheme(theme) {
        if (theme === 'system') {
            document.documentElement.setAttribute('data-theme', systemDarkQuery.matches ? 'dark' : 'light');
        } else {
            document.documentElement.setAttribute('data-theme', theme);
        }
        localStorage.setItem('dailyflow-theme', theme);
        updateThemeIcon(theme);
    }

    systemDarkQuery.addEventListener('change', () => {
        if (localStorage.getItem('dailyflow-theme') === 'system') applyTheme('system');
    });

    let savedTheme = localStorage.getItem('dailyflow-theme') || 'system';
    applyTheme(savedTheme);

    themeBtn.addEventListener('click', () => {
        let currentTheme = localStorage.getItem('dailyflow-theme') || 'system';
        let nextIndex = (THEMES.indexOf(currentTheme) + 1) % THEMES.length;
        applyTheme(THEMES[nextIndex]);
    });

    // Marked Renderer
    const renderer = new marked.Renderer();
    let currentMarkdownPath = '';
    renderer.image = function(token) {
        let href = token.href || '';
        const title = token.title || '';
        const text = token.text || '';
        if (href && !href.startsWith('http') && !href.startsWith('/')) {
            const dir = currentMarkdownPath.substring(0, currentMarkdownPath.lastIndexOf('/'));
            if (href.startsWith('./')) href = href.substring(2);
            href = `/raw${dir}/${href}`;
        }
        return `<img src="${href}" alt="${text}" title="${title}">`;
    };
    marked.setOptions({ renderer });

    async function fetchPage(page) {
        if (isLoading || !hasMore || currentMode === 'search') return;
        isLoading = true;
        try {
            const res = await fetch(`/api/list?page=${page}`);
            if (res.status === 401) { window.location.href = '/login.html'; return; }
            const entries = await res.json();
            if (!entries || entries.length === 0) {
                hasMore = false;
                sentinelEl.style.display = 'none';
                return;
            }
            renderEntries(entries);
            currentPage++;
        } catch (error) { console.error(error); }
        finally { isLoading = false; }
    }

    async function performSearch(query) {
        currentMode = 'search';
        currentSearchQuery = query;
        timelineEl.innerHTML = '';
        sentinelEl.style.display = 'none';
        
        try {
            const res = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
            if (res.status === 401) { window.location.href = '/login.html'; return; }
            const entries = await res.json();
            if (!entries || entries.length === 0) {
                timelineEl.innerHTML = '<p class="empty-msg">No results found.</p>';
            } else {
                renderEntries(entries);
            }
        } catch (error) { console.error(error); }
    }

    function renderEntries(entries) {
        entries.forEach(entry => renderTimelineItem(entry.path, entry.content));
    }

    function renderTimelineItem(path, markdown) {
        const dateStr = path.split('/').pop().replace('.md', '');
        const itemEl = document.createElement('div');
        itemEl.className = 'timeline-item';
        
        const cardEl = document.createElement('div');
        cardEl.className = 'timeline-card';
        
        const contentWrapper = document.createElement('div');
        contentWrapper.className = 'markdown-content';
        
        currentMarkdownPath = path;
        const lines = markdown.split('\n');
        const needsExpansion = lines.filter(l => l.trim() !== '').length > 4;

        if (needsExpansion) {
            contentWrapper.innerHTML = marked.parse(lines.slice(0, 5).join('\n'));
            const btn = document.createElement('button');
            btn.className = 'read-more-btn';
            btn.textContent = 'Read More';
            btn.onclick = () => {
                currentMarkdownPath = path;
                contentWrapper.innerHTML = marked.parse(markdown);
                btn.remove();
            };
            cardEl.appendChild(contentWrapper);
            cardEl.appendChild(btn);
        } else {
            contentWrapper.innerHTML = marked.parse(markdown);
            cardEl.appendChild(contentWrapper);
        }

        itemEl.innerHTML = `<div class="timeline-date">${dateStr}</div>`;
        itemEl.appendChild(cardEl);
        timelineEl.appendChild(itemEl);
    }


    // Search input handler
    let searchTimeout;
    searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        const query = e.target.value.trim();
        if (query === '') {
            currentMode = 'timeline';
            timelineEl.innerHTML = '';
            currentPage = 1;
            hasMore = true;
            sentinelEl.style.display = 'flex';
            fetchPage(1);
        } else {
            searchTimeout = setTimeout(() => performSearch(query), 500);
        }
    });

    // Infinite scroll
    const observer = new IntersectionObserver((entries) => {
        if (entries[0].isIntersecting && !isLoading && hasMore && currentMode === 'timeline') {
            fetchPage(currentPage);
        }
    }, { rootMargin: '200px' });
    observer.observe(sentinelEl);
});
