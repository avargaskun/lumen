<?php

namespace Illuminate\Support;

use Illuminate\Contracts\Foundation\Application;

abstract class ServiceProvider
{
    /**
     * The application instance.
     *
     * @var Application
     */
    protected $app;

    /**
     * All of the registered booting callbacks.
     *
     * @var array
     */
    protected $bootingCallbacks = [];

    /**
     * All of the registered booted callbacks.
     *
     * @var array
     */
    protected $bootedCallbacks = [];

    /**
     * Create a new service provider instance.
     */
    public function __construct(Application $app)
    {
        $this->app = $app;
    }

    /**
     * Register any application services.
     */
    abstract public function register(): void;

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        //
    }

    /**
     * Register a booting callback to be run before the "boot" method is called.
     */
    public function booting(callable $callback): void
    {
        $this->bootingCallbacks[] = $callback;
    }

    /**
     * Register a booted callback to be run after the "boot" method is called.
     */
    public function booted(callable $callback): void
    {
        $this->bootedCallbacks[] = $callback;
    }

    /**
     * Call the registered booting callbacks.
     */
    public function callBootingCallbacks(): void
    {
        foreach ($this->bootingCallbacks as $callback) {
            $this->app->call($callback);
        }
    }

    /**
     * Call the registered booted callbacks.
     */
    public function callBootedCallbacks(): void
    {
        foreach ($this->bootedCallbacks as $callback) {
            $this->app->call($callback);
        }
    }

    /**
     * Get the services provided by the provider.
     *
     * @return array<int, string>
     */
    public function provides(): array
    {
        return [];
    }

    /**
     * Determine if the provider is deferred.
     */
    public function isDeferred(): bool
    {
        return false;
    }

    /**
     * Get the default providers for a Laravel application.
     *
     * @return array<int, class-string>
     */
    public static function defaultProviders(): array
    {
        return [
            \Illuminate\Auth\AuthServiceProvider::class,
            \Illuminate\Cache\CacheServiceProvider::class,
            \Illuminate\Database\DatabaseServiceProvider::class,
            \Illuminate\Filesystem\FilesystemServiceProvider::class,
            \Illuminate\Queue\QueueServiceProvider::class,
        ];
    }
}
