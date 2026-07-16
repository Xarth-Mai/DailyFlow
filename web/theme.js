document.addEventListener('DOMContentLoaded', () => {
    const themeBtn = document.getElementById('theme-btn');
    const themeIcon = document.getElementById('theme-icon');
    if (!themeBtn || !themeIcon) return;

    const themes = ['system', 'light', 'dark'];
    const systemDarkQuery = window.matchMedia('(prefers-color-scheme: dark)');

    function updateThemeIcon(theme) {
        if (theme === 'system') {
            themeIcon.innerHTML = `<path d="M21 9C19.3529 9 17.8917 9.79647 16.9808 11.0253M21 6C18.5797 6 16.4104 7.07479 14.9434 8.77313M21 3C17.1326 3 13.7313 4.99586 11.77 8.01376M11.77 8.01376C9.72698 8.16181 8.00348 9.48869 7.25 11.25C4.7 11.6562 3 13.7572 3 16.0315C3 18.7755 5.28335 21 8.1 21L15.75 21C18.0972 21 20 19.1279 20 16.8185C20 15.1039 18.951 13.5202 17.45 12.875C17.4116 12.2181 17.2475 11.5941 16.9808 11.0253M11.77 8.01376C11.8958 8.00465 12.0229 8 12.151 8C13.1755 8 14.1323 8.28298 14.9434 8.77313M14.9434 8.77313C15.8305 9.30914 16.5435 10.0929 16.9808 11.0253"/>`;
        } else if (theme === 'light') {
            themeIcon.innerHTML = '<circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>';
        } else {
            themeIcon.innerHTML = '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>';
        }
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute(
            'data-theme',
            theme === 'system' ? (systemDarkQuery.matches ? 'dark' : 'light') : theme,
        );
        localStorage.setItem('dailyflow-theme', theme);
        updateThemeIcon(theme);
    }

    systemDarkQuery.addEventListener('change', () => {
        if (localStorage.getItem('dailyflow-theme') === 'system') applyTheme('system');
    });
    applyTheme(localStorage.getItem('dailyflow-theme') || 'system');
    themeBtn.addEventListener('click', () => {
        const currentTheme = localStorage.getItem('dailyflow-theme') || 'system';
        applyTheme(themes[(themes.indexOf(currentTheme) + 1) % themes.length]);
    });
});
