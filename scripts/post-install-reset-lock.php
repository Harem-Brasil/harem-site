<?php

/**
 * Repõe composer.lock ao último commit após o Composer alterar o ficheiro no deploy.
 * Não deve falhar o Composer: qualquer erro é ignorado.
 */

$root = dirname(__DIR__);

try {
    if (!is_dir($root . DIRECTORY_SEPARATOR . '.git')) {
        exit(0);
    }
    if (!function_exists('exec')) {
        exit(0);
    }

    $cmd = sprintf('git -C %s checkout HEAD -- composer.lock', escapeshellarg($root));
    @exec($cmd, $unused, $unusedCode);
} catch (Throwable $e) {
    // intencionalmente vazio
}

exit(0);
