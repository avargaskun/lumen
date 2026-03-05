<?php

namespace Illuminate\Container;

use Illuminate\Contracts\Container\Container;
use Illuminate\Contracts\Container\ContextualBindingBuilder as ContextualBindingBuilderContract;

class ContextualBindingBuilder implements ContextualBindingBuilderContract
{
    /**
     * The underlying container instance.
     *
     * @var Container
     */
    protected $container;

    /**
     * The concrete instance(s) that the contextual binding applies to.
     *
     * @var string|array
     */
    protected $concrete;

    /**
     * The abstract target that the contextual binding should resolve.
     *
     * @var string
     */
    protected $needs;

    /**
     * Create a new contextual binding builder.
     */
    public function __construct(Container $container, $concrete)
    {
        $this->container = $container;
        $this->concrete = $concrete;
    }

    /**
     * Define the abstract target that the contextual binding should resolve.
     *
     * @param  string  $abstract
     * @return $this
     */
    public function needs(string $abstract): self
    {
        $this->needs = $abstract;

        return $this;
    }

    /**
     * Define the implementation for the contextual binding.
     *
     * @param  \Closure|string|array  $implementation
     * @return void
     */
    public function give($implementation): void
    {
        foreach ($this->normalizeConcrete() as $concrete) {
            $this->container->addContextualBinding(
                $concrete,
                $this->needs,
                $implementation
            );
        }
    }

    /**
     * Define tagged services to be used as the implementation for the contextual binding.
     *
     * @param  string  $tag
     * @return void
     */
    public function giveTagged(string $tag): void
    {
        $this->give(function ($container) use ($tag) {
            $taggedServices = $container->tagged($tag);

            return is_array($taggedServices) ? $taggedServices : iterator_to_array($taggedServices);
        });
    }

    /**
     * Specify the configuration item to bind as a primitive.
     *
     * @param  string  $key
     * @param  mixed  $default
     * @return void
     */
    public function giveConfig(string $key, $default = null): void
    {
        $this->give(function ($container) use ($key, $default) {
            return $container->make('config')->get($key, $default);
        });
    }

    /**
     * Normalize the concrete bindings into an array.
     *
     * @return array
     */
    protected function normalizeConcrete(): array
    {
        return is_array($this->concrete) ? $this->concrete : [$this->concrete];
    }
}
