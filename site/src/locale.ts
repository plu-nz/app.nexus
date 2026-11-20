import { readSharedPreference, writeSharedPreference } from './locale-preference';

export type Locale = 'en' | 'pt' | 'es';
export type Preference = Locale | 'system';
const key = 'plunz.nexus-site.locale';

export function detectLocale(tags: readonly string[] = typeof navigator === 'undefined' ? [] : navigator.languages): Locale {
  const preferred = tags.length ? tags : typeof navigator === 'undefined' ? [] : [navigator.language];
  for (const tag of preferred) {
    const base = tag.trim().toLowerCase().split(/[-_]/)[0];
    if (base === 'en' || base === 'pt' || base === 'es') return base;
  }
  return 'en';
}

export function readPreference(): Preference {
  try {
    return readSharedPreference(localStorage, key);
  } catch { return 'system'; }
}

export function savePreference(value: Preference): void {
  try {
    writeSharedPreference(localStorage, key, value);
  } catch { /* Private browsing can disable storage. */ }
}

const en = {
  skip: 'Skip to content', language: 'Language', system: 'System language',
  navIdea: 'The idea', navFeatures: 'Features', navInstall: 'Download',
  title: 'Nexus — Your TV, one button away',
  description: 'Nexus is a TV panel for Linux: apps, volume, brightness, and a sleep timer, one remote button away. Download it for Linux.',
  eyebrow: 'LINUX TV PANEL',
  headlineStart: 'Your TV,', headlineEnd: 'one button away.',
  lead: 'Nexus slides up over whatever is playing and puts your TV apps, volume, audio output, brightness, and a sleep timer in one row you drive with a remote.',
  download: 'Download for Linux', downloadMeta: 'x86_64 · tar.gz · latest release', source: 'Source on GitHub',
  mockVolume: 'Volume', mockOutput: 'Output', mockBrightness: 'Brightness', mockSleep: 'Sleep', mockSleepValue: '45 min',
  ideaEyebrow: 'THE IDEA',
  ideaTitle: 'A home button for a Linux TV.',
  ideaBody: 'A PC on a TV has no system menu made for the couch: turning the volume down or setting a sleep timer means reaching for a keyboard. Nexus is that menu. Press one button on the remote, change what you need, and get back to the show.',
  pillars: [
    { title: 'One button', body: 'Bind nexus-toggle to a key or a remote button. The first press starts Nexus; later presses show and hide the panel.' },
    { title: 'Over the picture', body: 'On compositors with the layer shell, the translucent panel covers fullscreen video, so what is playing stays visible behind it.' },
    { title: 'Always ready', body: 'Nexus keeps running while hidden, so the sleep timer keeps counting after the panel closes.' },
  ],
  featuresEyebrow: 'FEATURES', featuresTitle: 'Everything the couch needs.',
  features: [
    'Opens Lounge and VacuumTube when they are installed. On Hyprland, an app that is already open comes to the front instead of starting again.',
    'Volume in 5% steps and mute, plus audio output switching when there are two or more outputs.',
    'Brightness in 10% steps through brightnessctl or ddcutil.',
    'A sleep timer from 15 minutes to four hours, or suspend right away.',
    'A hidden Close apps action that closes every open TV app.',
    'Follows your GTK theme, in English, Portuguese, or Spanish.',
  ],
  controlsTitle: 'Controls',
  controls: [
    ['Left / Right', 'Move between apps and controls'],
    ['Up / Down', 'Adjust volume, output, brightness, or the sleep timer'],
    ['Enter / OK', 'Open an app, mute, or confirm'],
    ['Back / Escape', 'Close the panel'],
  ],
  installEyebrow: 'DOWNLOAD', installTitle: 'Install Nexus.',
  installLead: 'Nexus is built for Arch Linux and Omarchy on x86_64. On Wayland compositors with the layer shell, such as Hyprland, Sway, and KDE, it appears over fullscreen video; elsewhere it opens as a normal window.',
  steps: [
    { title: 'Install the libraries it uses', code: 'sudo pacman -S --needed gtk4 webkitgtk-6.0 gtk4-layer-shell brightnessctl' },
    { title: 'Download and extract the archive', code: 'tar -xzf nexus-linux-x86_64.tar.gz' },
    { title: 'Run the installer', code: 'cd nexus-*/ && ./install.sh' },
    { title: 'Bind this command to a key or a remote button', code: 'nexus-toggle' },
  ],
  installNote: 'The installer puts nexus and nexus-toggle in ~/.local/bin, with a desktop entry and an icon, for your user only. Audio needs PipeWire with WirePlumber, brightness needs brightnessctl or ddcutil, and power uses systemd. A control whose tool is missing shows as unavailable.',
  copy: 'Copy', copied: 'Copied',
  checksum: 'SHA-256 checksum', releases: 'All releases', fromSource: 'Build from source',
  pairEyebrow: 'WORKS WELL WITH', pairBody: 'Lounge, a TV app for Stremio addons that Nexus opens from the same row.', pairLink: 'Meet Lounge',
  disclaimer: 'Nexus is an independent app. VacuumTube and the other apps it opens are separate projects with their own terms.',
  partOf: 'A PLUNZ app', privacy: 'Privacy', terms: 'Terms',
};

export type Copy = typeof en;

export const copy: Record<Locale, Copy> = {
  en,
  pt: {
    skip: 'Pular para o conteúdo', language: 'Idioma', system: 'Idioma do sistema',
    navIdea: 'A ideia', navFeatures: 'Recursos', navInstall: 'Download',
    title: 'Nexus — Sua TV a um botão de distância',
    description: 'Nexus é um painel de TV para Linux: apps, volume, brilho e timer de desligamento a um botão do controle. Baixe para Linux.',
    eyebrow: 'PAINEL DE TV PARA LINUX',
    headlineStart: 'Sua TV,', headlineEnd: 'a um botão.',
    lead: 'O Nexus surge sobre o que estiver passando e reúne seus apps de TV, volume, saída de áudio, brilho e timer de desligamento em uma única fileira que você usa com o controle remoto.',
    download: 'Baixar para Linux', downloadMeta: 'x86_64 · tar.gz · última versão', source: 'Código no GitHub',
    mockVolume: 'Volume', mockOutput: 'Saída', mockBrightness: 'Brilho', mockSleep: 'Timer', mockSleepValue: '45 min',
    ideaEyebrow: 'A IDEIA',
    ideaTitle: 'Um botão de início para uma TV com Linux.',
    ideaBody: 'Um PC ligado à TV não tem um menu de sistema feito para o sofá: baixar o volume ou programar o desligamento exige pegar o teclado. O Nexus é esse menu. Aperte um botão do controle, ajuste o que precisar e volte para o filme.',
    pillars: [
      { title: 'Um botão', body: 'Associe o nexus-toggle a uma tecla ou a um botão do controle. O primeiro toque abre o Nexus; os seguintes mostram e escondem o painel.' },
      { title: 'Sobre a imagem', body: 'Em compositores com layer shell, o painel translúcido fica sobre o vídeo em tela cheia, e o que está passando continua visível atrás dele.' },
      { title: 'Sempre pronto', body: 'O Nexus continua rodando escondido, então o timer de desligamento segue contando depois que o painel fecha.' },
    ],
    featuresEyebrow: 'RECURSOS', featuresTitle: 'Tudo o que o sofá precisa.',
    features: [
      'Abre o Lounge e o VacuumTube quando estão instalados. No Hyprland, um app já aberto vem para a frente em vez de abrir de novo.',
      'Volume de 5% em 5% e mudo, além de troca de saída de áudio quando há duas ou mais saídas.',
      'Brilho de 10% em 10% com brightnessctl ou ddcutil.',
      'Timer de desligamento de 15 minutos a quatro horas, ou suspensão imediata.',
      'Uma ação escondida de Fechar apps, que fecha todos os apps de TV abertos.',
      'Segue o tema GTK, em inglês, português ou espanhol.',
    ],
    controlsTitle: 'Controles',
    controls: [
      ['Esquerda / Direita', 'Andar entre apps e controles'],
      ['Cima / Baixo', 'Ajustar volume, saída, brilho ou timer'],
      ['Enter / OK', 'Abrir um app, ativar o mudo ou confirmar'],
      ['Voltar / Esc', 'Fechar o painel'],
    ],
    installEyebrow: 'DOWNLOAD', installTitle: 'Instale o Nexus.',
    installLead: 'O Nexus é feito para Arch Linux e Omarchy em x86_64. Em compositores Wayland com layer shell, como Hyprland, Sway e KDE, ele aparece sobre o vídeo em tela cheia; nos demais, abre como uma janela comum.',
    steps: [
      { title: 'Instale as bibliotecas que ele usa', code: 'sudo pacman -S --needed gtk4 webkitgtk-6.0 gtk4-layer-shell brightnessctl' },
      { title: 'Baixe e extraia o arquivo', code: 'tar -xzf nexus-linux-x86_64.tar.gz' },
      { title: 'Rode o instalador', code: 'cd nexus-*/ && ./install.sh' },
      { title: 'Associe este comando a uma tecla ou botão do controle', code: 'nexus-toggle' },
    ],
    installNote: 'O instalador coloca o nexus e o nexus-toggle em ~/.local/bin, com atalho e ícone, só para o seu usuário. O áudio precisa do PipeWire com WirePlumber, o brilho precisa de brightnessctl ou ddcutil, e as opções de energia usam o systemd. Um controle sem a ferramenta necessária aparece como indisponível.',
    copy: 'Copiar', copied: 'Copiado',
    checksum: 'Checksum SHA-256', releases: 'Todas as versões', fromSource: 'Compilar do código',
    pairEyebrow: 'COMBINA COM', pairBody: 'Lounge, um app de TV para addons do Stremio que o Nexus abre pela mesma fileira.', pairLink: 'Conheça o Lounge',
    disclaimer: 'O Nexus é um app independente. O VacuumTube e os outros apps que ele abre são projetos separados, com termos próprios.',
    partOf: 'Um app PLUNZ', privacy: 'Privacidade', terms: 'Termos',
  },
  es: {
    skip: 'Ir al contenido', language: 'Idioma', system: 'Idioma del sistema',
    navIdea: 'La idea', navFeatures: 'Funciones', navInstall: 'Descarga',
    title: 'Nexus — Tu TV a un botón de distancia',
    description: 'Nexus es un panel de TV para Linux: apps, volumen, brillo y temporizador de apagado a un botón del control. Descárgalo para Linux.',
    eyebrow: 'PANEL DE TV PARA LINUX',
    headlineStart: 'Tu TV,', headlineEnd: 'a un botón.',
    lead: 'Nexus aparece sobre lo que se esté reproduciendo y reúne tus apps de TV, el volumen, la salida de audio, el brillo y un temporizador de apagado en una sola fila que manejas con el control remoto.',
    download: 'Descargar para Linux', downloadMeta: 'x86_64 · tar.gz · última versión', source: 'Código en GitHub',
    mockVolume: 'Volumen', mockOutput: 'Salida', mockBrightness: 'Brillo', mockSleep: 'Apagado', mockSleepValue: '45 min',
    ideaEyebrow: 'LA IDEA',
    ideaTitle: 'Un botón de inicio para una TV con Linux.',
    ideaBody: 'Un PC conectado a la TV no tiene un menú de sistema pensado para el sofá: bajar el volumen o programar el apagado obliga a tomar el teclado. Nexus es ese menú. Pulsa un botón del control, ajusta lo que necesites y vuelve a la película.',
    pillars: [
      { title: 'Un botón', body: 'Asigna nexus-toggle a una tecla o a un botón del control. La primera pulsación abre Nexus; las siguientes muestran y ocultan el panel.' },
      { title: 'Sobre la imagen', body: 'En compositores con layer shell, el panel translúcido cubre el video a pantalla completa y lo que se reproduce sigue visible detrás.' },
      { title: 'Siempre listo', body: 'Nexus sigue en ejecución mientras está oculto, así que el temporizador de apagado sigue contando después de cerrar el panel.' },
    ],
    featuresEyebrow: 'FUNCIONES', featuresTitle: 'Todo lo que el sofá necesita.',
    features: [
      'Abre Lounge y VacuumTube cuando están instalados. En Hyprland, una app ya abierta pasa al frente en lugar de iniciarse otra vez.',
      'Volumen de 5 % en 5 % y silencio, además de cambio de salida de audio cuando hay dos o más salidas.',
      'Brillo de 10 % en 10 % con brightnessctl o ddcutil.',
      'Temporizador de apagado de 15 minutos a cuatro horas, o suspensión inmediata.',
      'Una acción oculta de Cerrar apps, que cierra todas las apps de TV abiertas.',
      'Sigue tu tema GTK, en inglés, portugués o español.',
    ],
    controlsTitle: 'Controles',
    controls: [
      ['Izquierda / Derecha', 'Moverse entre apps y controles'],
      ['Arriba / Abajo', 'Ajustar volumen, salida, brillo o temporizador'],
      ['Enter / OK', 'Abrir una app, silenciar o confirmar'],
      ['Atrás / Esc', 'Cerrar el panel'],
    ],
    installEyebrow: 'DESCARGA', installTitle: 'Instala Nexus.',
    installLead: 'Nexus está hecho para Arch Linux y Omarchy en x86_64. En compositores Wayland con layer shell, como Hyprland, Sway y KDE, aparece sobre el video a pantalla completa; en los demás se abre como una ventana normal.',
    steps: [
      { title: 'Instala las bibliotecas que usa', code: 'sudo pacman -S --needed gtk4 webkitgtk-6.0 gtk4-layer-shell brightnessctl' },
      { title: 'Descarga y extrae el archivo', code: 'tar -xzf nexus-linux-x86_64.tar.gz' },
      { title: 'Ejecuta el instalador', code: 'cd nexus-*/ && ./install.sh' },
      { title: 'Asigna este comando a una tecla o a un botón del control', code: 'nexus-toggle' },
    ],
    installNote: 'El instalador coloca nexus y nexus-toggle en ~/.local/bin, con un acceso directo y un icono, solo para tu usuario. El audio necesita PipeWire con WirePlumber, el brillo necesita brightnessctl o ddcutil, y las opciones de energía usan systemd. Un control sin su herramienta aparece como no disponible.',
    copy: 'Copiar', copied: 'Copiado',
    checksum: 'Suma SHA-256', releases: 'Todas las versiones', fromSource: 'Compilar desde el código',
    pairEyebrow: 'VA BIEN CON', pairBody: 'Lounge, una app de TV para addons de Stremio que Nexus abre desde la misma fila.', pairLink: 'Conoce Lounge',
    disclaimer: 'Nexus es una app independiente. VacuumTube y las demás apps que abre son proyectos separados, con sus propios términos.',
    partOf: 'Una app de PLUNZ', privacy: 'Privacidad', terms: 'Términos',
  },
};
