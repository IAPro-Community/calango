import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import {
  ApprovalCard,
  Badge,
  Button,
  Card,
  Field,
  Input,
  NavItem,
  NodePill,
  PageHeader,
  Select,
  StatCard,
} from '@/index';

describe('Button', () => {
  it('is a non-submitting button by default and calls onClick', async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Aprovar ação</Button>);
    const button = screen.getByRole('button', { name: 'Aprovar ação' });
    expect(button).toHaveAttribute('type', 'button');
    await userEvent.click(button);
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('applies the variant and keeps caller classes last', () => {
    render(
      <Button variant="danger" className="w-full">
        Parar
      </Button>,
    );
    const button = screen.getByRole('button', { name: 'Parar' });
    expect(button.className).toContain('text-danger-text');
    expect(button.className).toContain('w-full');
  });

  it('renders the child element with asChild', () => {
    render(
      <Button asChild variant="ghost">
        <a href="#detalhes">Ver detalhes</a>
      </Button>,
    );
    const link = screen.getByRole('link', { name: 'Ver detalhes' });
    expect(link.className).toContain('border-line');
  });

  it('does not fire when disabled', async () => {
    const onClick = vi.fn();
    render(
      <Button disabled onClick={onClick}>
        Aprovar
      </Button>,
    );
    await userEvent.click(screen.getByRole('button', { name: 'Aprovar' }));
    expect(onClick).not.toHaveBeenCalled();
  });
});

describe('Badge', () => {
  it('maps each tone to its colour pair', () => {
    render(<Badge tone="warn">! aprovação pendente</Badge>);
    expect(screen.getByText('! aprovação pendente').className).toContain('text-warn-text');
  });
});

describe('NodePill', () => {
  it('shows the node name and state, and greys the dot when offline', () => {
    const { container } = render(<NodePill kind="pc" state="offline" label="Seu computador" status="offline · 14m" />);
    expect(screen.getByText('Seu computador')).toBeInTheDocument();
    expect(screen.getByText('offline · 14m').className).toContain('text-ink-soft');
    expect(container.querySelector('[aria-hidden="true"]')?.className).toContain('bg-line');
  });
});

describe('NavItem', () => {
  it('marks the active item as the current page and shows the counter', () => {
    render(
      <>
        <NavItem>Conversas</NavItem>
        <NavItem active count={3}>
          Para você
        </NavItem>
      </>,
    );
    expect(screen.getByRole('button', { name: /Para você/ })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: 'Conversas' })).not.toHaveAttribute('aria-current');
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('keeps the label as the button name and describes the counter', () => {
    render(
      <NavItem count={2} countLabel="2 pendentes">
        Aprovações
      </NavItem>,
    );
    const button = screen.getByRole('button', { name: 'Aprovações' });
    expect(button).toHaveAccessibleDescription('2 pendentes');
  });

  it('hides the counter at zero', () => {
    render(
      <NavItem count={0} countLabel="0 pendentes">
        Aprovações
      </NavItem>,
    );
    expect(screen.getByRole('button', { name: 'Aprovações' })).not.toHaveAttribute('aria-describedby');
    expect(screen.queryByText('0')).not.toBeInTheDocument();
  });

  it('hides a zero counter', () => {
    render(<NavItem count={0}>Conversas</NavItem>);
    expect(screen.queryByText('0')).not.toBeInTheDocument();
  });
});

describe('ApprovalCard', () => {
  it('shows actions while pending and hides the receipt', () => {
    render(
      <ApprovalCard
        state="pending"
        eyebrow="Aprovação pendente"
        title="Enviar e-mail"
        detail="email.send · sha256:9f82c1"
        receipt="não deve aparecer"
        actions={<Button>Aprovar ação</Button>}
      />,
    );
    expect(screen.getByRole('heading', { name: 'Enviar e-mail' })).toBeInTheDocument();
    expect(screen.getByText('email.send · sha256:9f82c1')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Aprovar ação' })).toBeInTheDocument();
    expect(screen.queryByText('não deve aparecer')).not.toBeInTheDocument();
  });

  it('shows the receipt and no actions once decided', () => {
    render(
      <ApprovalCard
        state="failed"
        eyebrow="Falhou após aprovação"
        title="Publicar release"
        detail="gh.release.create"
        receipt="Recibo: failed"
        actions={<Button>Aprovar ação</Button>}
      />,
    );
    expect(screen.getByText('Recibo: failed').className).toContain('text-danger-text');
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });
});

describe('StatCard', () => {
  it('renders label, value and hint', () => {
    render(<StatCard label="Gasto mensal" value="R$ 18,40" hint="36% do limite" hintTone="warn" />);
    expect(screen.getByRole('heading', { name: 'Gasto mensal' })).toBeInTheDocument();
    expect(screen.getByText('R$ 18,40')).toBeInTheDocument();
    expect(screen.getByText('36% do limite').className).toContain('text-warn-text');
  });
});

describe('Card', () => {
  it('names the region by its title', () => {
    render(
      <Card title="Criar rotina" aside={<Badge>ativa</Badge>}>
        corpo
      </Card>,
    );
    expect(screen.getByRole('region', { name: 'Criar rotina' })).toHaveTextContent('corpo');
    expect(screen.getByText('ativa')).toBeInTheDocument();
  });

  it('has no heading or name without a title', () => {
    render(<Card>só corpo</Card>);
    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
    expect(screen.queryByRole('region')).not.toBeInTheDocument();
  });
});

describe('Field', () => {
  it('labels the control it wraps', async () => {
    render(
      <Field label="Nome" hint="Como aparece no menu">
        <Input />
      </Field>,
    );
    const input = screen.getByLabelText(/Nome/);
    await userEvent.type(input, 'Ana');
    expect(input).toHaveValue('Ana');
    expect(screen.getByText('Como aparece no menu')).toBeInTheDocument();
  });

  it('works with Select', async () => {
    render(
      <Field label="Disparo">
        <Select defaultValue="cron">
          <option value="cron">Cron</option>
          <option value="interval">Intervalo</option>
        </Select>
      </Field>,
    );
    await userEvent.selectOptions(screen.getByLabelText('Disparo'), 'interval');
    expect(screen.getByLabelText('Disparo')).toHaveValue('interval');
  });
});

describe('PageHeader', () => {
  it('renders the page heading, meta line and actions', () => {
    render(<PageHeader title="Conversas" meta="nó: Auto (PC)" actions={<Button>Nova conversa</Button>} />);
    expect(screen.getByRole('heading', { level: 1, name: 'Conversas' })).toBeInTheDocument();
    expect(screen.getByText('nó: Auto (PC)')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Nova conversa' })).toBeInTheDocument();
  });
});
