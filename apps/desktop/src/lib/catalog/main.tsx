import { StrictMode, useState } from 'react';
import { createRoot } from 'react-dom/client';
import '../../../../../packages/tokens/generated/tokens.css';
import '@/styles/theme.css';
import { ApprovalCard, Badge, Button, NavItem, NodePill, StatCard } from '@/index';

// Development catalog of the UI kit: `npx vite` in apps/desktop, then open /catalog.html.
// It is not part of the production build, which only bundles index.html.
function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-sm rounded-lg border border-dashed border-line p-md">
      <h2 className="m-0 font-mono text-[11px] font-bold tracking-[0.08em] text-moss-text uppercase">{title}</h2>
      <div className="flex flex-wrap items-start gap-sm">{children}</div>
    </section>
  );
}

function Catalog() {
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  const [active, setActive] = useState('para-voce');
  return (
    <div data-theme={theme} className="min-h-screen bg-bg p-2xl font-body text-ink">
      <header className="mb-lg flex items-center justify-between">
        <h1 className="m-0 font-display text-[28px] font-bold">Calango UI kit</h1>
        <Button variant="ghost" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>
          Tema: {theme === 'dark' ? 'escuro' : 'claro'}
        </Button>
      </header>
      <main className="flex flex-col gap-md">
        <Section title="Button">
          <Button>Aprovar ação</Button>
          <Button variant="postit">Novo agente</Button>
          <Button variant="ghost">Ver detalhes</Button>
          <Button variant="danger">Recusar / Parar</Button>
          <Button disabled>Desabilitado</Button>
        </Section>
        <Section title="Badge">
          <Badge tone="ok">● concluído</Badge>
          <Badge tone="warn">! aprovação pendente</Badge>
          <Badge tone="postit">PRO · 24/7</Badge>
          <Badge tone="neutral">08:00 · seg–sex</Badge>
          <Badge tone="danger">✕ falhou</Badge>
        </Section>
        <Section title="NodePill">
          <NodePill className="w-[236px]" kind="pc" state="online" label="Seu computador" status="ligado" />
          <NodePill className="w-[236px]" kind="pc" state="offline" label="Seu computador" status="offline · 14m" />
          <NodePill className="w-[236px]" kind="vps" state="online" label="VPS (nuvem)" status="pronta" />
          <NodePill className="w-[236px]" kind="vps" state="working" label="VPS (nuvem)" status="trabalhando" />
        </Section>
        <Section title="NavItem">
          <nav className="flex w-[212px] flex-col gap-2xs">
            <NavItem active={active === 'conversas'} onClick={() => setActive('conversas')}>
              Conversas
            </NavItem>
            <NavItem active={active === 'para-voce'} count={3} onClick={() => setActive('para-voce')}>
              Para você
            </NavItem>
          </nav>
        </Section>
        <Section title="ApprovalCard">
          <ApprovalCard
            className="w-[410px]"
            state="pending"
            eyebrow="Aprovação pendente · seq #1 · expira em 23h"
            title="Enviar e-mail para Marina (Ateliê Sol)"
            detail={'email.send · "Orçamento: identidade visual — R$ 4.800" · sha256:9f82c1'}
            actions={
              <>
                <Button>Aprovar ação</Button>
                <Button variant="ghost">Ver detalhes</Button>
                <Button variant="danger">Recusar / Parar</Button>
              </>
            }
          />
          <ApprovalCard
            className="w-[410px]"
            state="approved"
            eyebrow="✓ Concluído · aprovado pelo iPhone"
            title="Enviar e-mail para Marina (Ateliê Sol)"
            detail="email.send · entregue às 08:02 · audit_log #418"
            receipt="Recibo: mensagem enviada (250 OK) · snapshot verificado"
          />
          <ApprovalCard
            className="w-[410px]"
            state="failed"
            eyebrow="✕ Falhou após aprovação"
            title="Publicar release v0.2.0 no GitHub"
            detail="gh.release.create · erro: token sem escopo write:packages"
            receipt="Recibo: failed · nenhuma alteração publicada · retry disponível"
          />
        </Section>
        <Section title="StatCard">
          <StatCard
            className="w-[260px]"
            label="Gasto mensal · gateway"
            value="R$ 18,40"
            hint="36% do limite · 412k tokens"
          />
        </Section>
      </main>
    </div>
  );
}

const root = document.getElementById('root');
if (root === null) {
  throw new Error('missing #root');
}
createRoot(root).render(
  <StrictMode>
    <Catalog />
  </StrictMode>,
);
