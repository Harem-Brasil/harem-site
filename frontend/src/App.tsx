function App() {
  const particles = Array.from({ length: 9 }, (_, i) => ({
    left: `${(i + 1) * 10}%`,
    delay: `${[0, 2, 4, 1, 3, 5, 2, 4, 1][i]}s`,
  }))

  return (
    <div className="relative flex h-screen w-screen items-center justify-center overflow-hidden bg-black text-center">
      {/* Animated background */}
      <div
        className="animate-move-bg absolute inset-0 z-0"
        style={{
          width: '200%',
          height: '200%',
          background:
            'radial-gradient(circle at 30% 30%, rgba(212,175,55,0.08), transparent 40%), radial-gradient(circle at 70% 70%, rgba(212,175,55,0.05), transparent 50%)',
        }}
      />

      {/* Floating particles */}
      <div className="absolute inset-0 z-0 overflow-hidden">
        {particles.map((p, i) => (
          <span
            key={i}
            className="animate-float absolute block h-0.5 w-0.5 rounded-full"
            style={{
              left: p.left,
              animationDelay: p.delay,
              background: 'rgba(212,175,55,0.4)',
            }}
          />
        ))}
      </div>

      {/* Content */}
      <div className="relative z-10 max-w-[600px] px-8 py-8">
        <span className="mb-4 block text-sm font-bold tracking-widest text-gold">
          EXCLUSIVO PARA MAIORES DE 18 ANOS
        </span>

        <h1 className="mb-5 text-[42px] font-bold tracking-[2px] text-gold">
          Harem Brasil
        </h1>

        <p className="mb-8 text-lg leading-relaxed text-gray-300">
          Um novo espaço está sendo criado para quem busca conexões mais intensas, discretas e fora do comum.
          <br /><br />
          Conversas privadas, experiências exclusivas e liberdade total para explorar novas possibilidades sem julgamentos.
        </p>

        <p className="mb-8 text-lg leading-relaxed text-gray-300">
          Estamos abrindo as primeiras vagas antecipadas.
          <br />
          <strong className="text-white">Mulheres têm acesso prioritário à plataforma.</strong>
        </p>

        <a
          href="https://wa.me/5511919208046?text=Ol%C3%A1%20estou%20vindo%20do%20site%20e%20gostaria%20de%20saber%20mais%20sobre%20o%20H%C3%A1rem%20Brasil"
          className="inline-block rounded border-none bg-gradient-to-r from-gold to-gold-dark px-8 py-4 text-lg font-bold text-black shadow-[0_0_15px_rgba(212,175,55,0.3)] transition-all duration-300 hover:scale-105 hover:bg-gradient-to-r hover:from-gold-light hover:to-gold hover:shadow-[0_0_25px_rgba(212,175,55,0.6)]"
        >
          Quero entrar primeiro
        </a>

        <div className="mt-8 text-xs text-gray-600">
          Plataforma em desenvolvimento &bull; Lançamento em breve
        </div>
      </div>
    </div>
  )
}

export default App
